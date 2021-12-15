(ns termishare.main
  (:require [reagent.dom :as rd]
            [reagent.core :as r]
            [termishare.constants :as const]
            [termishare.env :refer [SERVER_URL]]
            [lambdaisland.uri :refer [uri]]
            ["xterm" :as xterm]
            [termishare.components.mui :refer [Button]]))

;;; ------------------------------ Utils ------------------------------
(defonce state
  (r/atom {:ws-conn       nil
           :peer-conn     nil
           :data-channels {}
           :term          nil}))

(defonce connection-id (str (random-uuid)))
(defonce text-encoder (js/TextEncoder.))
(defonce text-decoder (js/TextDecoder. "utf-8"))
(defonce terminal-id "terminal")

(defn send-when-connected
  "Send a message via a websocket connection, Retry if it fails"
  ([ws-conn msg]
   (js/console.log "Registered to send " (clj->js (assoc msg
                                                         :From connection-id
                                                         :To const/TERMISHARE_WEBSOCKET_HOST_ID)))
   (send-when-connected ws-conn msg 0 100))

  ([ws-conn msg n limit]
   (if (< n limit)
     (if (= (.-readyState ws-conn) 1)
       ;; TODO a better way to handle ID
       (do
        (js/console.log "sending out:" (clj->js msg))
        (.send ws-conn (js/JSON.stringify (clj->js (assoc msg
                                                          :From connection-id
                                                          :To const/TERMISHARE_WEBSOCKET_HOST_ID)))))
       (js/setTimeout (fn [] (send-when-connected ws-conn msg (inc n) limit)) 1))
     (js/console.log "Drop message due reached retry limits: " (clj->js msg)))))

(defn element-size
  [el]
  (when el
    {:width (.-offsetWidth el)
     :height (.-offsetHeight el)}))

(defn guess-new-font-size
  [new-cols new-rows target-size]
  (let [term           (:term @state)
        cur-cols       (.-cols term)
        cur-rows       (.-rows term)
        cur-font-size  (.getOption term "fontSize")
        xterm-size     (element-size (-> (js/document.getElementById terminal-id) (.querySelector "xterm-screen")))
        new-hfont-mulp (* (/ cur-cols new-cols) (/ (:width target-size) (:width xterm-size)))
        new-vfont-mulp (* (/ cur-rows new-rows) (/ (:height target-size) (:height xterm-size)))]
    (if (> new-hfont-mulp new-vfont-mulp)
      (int (Math/floor (* cur-font-size new-vfont-mulp)))
      (int (Math/floor (* cur-font-size new-hfont-mulp))))))

;;; ------------------------------ Web Socket ------------------------------
(defn websocket-onmessage
  [e]
  (let [msg  (-> e .-data js/JSON.parse)
        data (-> msg .-Data js/JSON.parse)]

    ;; only handle messages that are sent by the host to us
    (when (and (= connection-id (.-To msg))
               (= const/TERMISHARE_WEBSOCKET_HOST_ID (.-From msg)))
      (condp = (keyword (.-Type msg))

        const/TRTCWillYouMarryMe
        (js/console.log "We shouldn't received this question, we should be the one who asks that")

        const/TRTCYes
        (.setRemoteDescription (:peer-conn @state) data)

        const/TRTCKiss
        (->> data
             js/RTCIceCandidate.
             (.addIceCandidate (:peer-conn @state)))

        :else
        (js/console.error "Unhandeled message type: " (.-Type msg))))))

(defn websocket-onclose
  [_e]
  (js/console.log "Websocket closed!")
  (swap! state assoc :ws-conn nil))

(defn ws-connect
  "Connect to websocket server"
  [url]
  (when-not (:ws-conn @state)
    (let [conn (js/WebSocket. url)]
      (set! (.-onopen conn) (fn [_e] (js/console.log "Websocket Connected")))
      (set! (.-onmessage conn) websocket-onmessage)
      (set! (.-onclose conn) websocket-onclose)
      (set! (.-onerror conn) websocket-onclose)
      (swap! state assoc :ws-conn conn))))

;;; ------------------------------ WebRTC ------------------------------
(def ice-candidate-config (clj->js {:iceServers [{:urls ["stun:stun1.l.google.com:19302"
                                                         "stun:stun2.l.google.com:19302"]}]
                                    :iceCandidatePoolSize 10}))

(defn rtc-onicecandidate
  [e]
  (when  (.-candidate e)
    (send-when-connected (:ws-conn @state)
                         {:Type const/TRTCKiss
                          :Data (-> e .-candidate .toJSON js/JSON.stringify)})))

(defn rtc-ondatachannel
  [e]
  (let [channel (.-channel e)]
    (set! (.-onclose channel) (fn [] (js/console.log "Channel " (.-label channel) " closed")))
    (set! (.-onopen channel) (fn [] (js/console.log "Channel " (.-label channel) " opened")))
    (set! (.-onmessage channel) (fn [e] (js/console.log "Recevied a message from channel: "
                                                        (.-label channel) " " (.-data e))))))

(defn rtc-on-termishare-channel
  [e]
  (let [data (-> e .-data js/Uint8Array.)]
    (.writeUtf8 (:term @state) data)))

(defn rtc-on-config-channel
  [e]
  (let [msg (->> e .-data (.decode text-decoder) js/JSON.parse)
        msg (js->clj msg :keywordize-keys true)]

    (condp = (-> msg :Type keyword)
      const/TTermWinsize
      (let [ws (:Data msg)
            term (:term @state)]
        (.setOption term (guess-new-font-size (.-cols term) (.-rows term)
                                              (element-size (js/document.getElementById terminal-id))))
        (.resize term (:Cols ws) (:Rows ws)))

      (js/console.log "I don't know you: " (clj->js msg)))))

(defn peer-connect
  []
  (let [conn               (js/RTCPeerConnection. ice-candidate-config)
        termishare-channel (.createDataChannel conn (str const/TERMISHARE_WEBRTC_DATA_CHANNEL))
        config-channel     (.createDataChannel conn (str const/TERMISHARE_WEBRTC_CONFIG_CHANNEL))]
    (set! (.-onconnectionstatechange conn) (fn [e] (js/console.log "Peer connection state change: " (clj->js e))))
    (set! (.-onicecandidate conn) rtc-onicecandidate)
    (set! (.-ondatachannel conn) rtc-ondatachannel)
    (set! (.-binaryType termishare-channel) "arraybuffer")
    (set! (.-binaryType config-channel) "arraybuffer")
    (set! (.-onmessage termishare-channel) rtc-on-termishare-channel)
    (set! (.-onmessage config-channel) rtc-on-config-channel)
    (swap! state assoc-in [:data-channels :termishare] termishare-channel)
    (swap! state assoc-in [:data-channels :config] config-channel)
    (swap! state assoc :peer-conn conn)))

(defn send-offer
  []
  (-> (:peer-conn @state)
      .createOffer
      (.then (fn [offer]
               (.setLocalDescription (:peer-conn @state) offer)
               (send-when-connected (:ws-conn @state) {:Type const/TRTCWillYouMarryMe
                                                       :Data (js/JSON.stringify offer)})))
      (.catch (fn [e]
                (js/console.log "Failed to send offer " e)))))


;;; ------------------------------ Component ------------------------------


(defn App
  []
  (r/create-class
   {:component-did-mount
    (fn []
      (let [term      (xterm/Terminal. #js {:cursorBlink  true
                                            :scrollback   1000
                                            :disableStdin false})]

        ;; Write back to the host
        (.onData term (fn [data] (when-let [channel (-> @state :data-channels :termishare)]
                                   (.send channel (.encode text-encoder data)))))
        (.open term (js/document.getElementById terminal-id))
        (swap! state assoc :term term)

        ; connect
        #_(ws-connect "ws://localhost:3000/ws")
        #_(peer-connect)
        #_(send-offer)))

    :reagent-render
    (fn []
      [:<>
       [:div {:id terminal-id :class "w-screen h-screen fixed top-0 left-0"}]
       [Button {:on-click (fn [_e]
                            (js/console.log "HEY: " SERVER_URL)
                            (ws-connect (str (assoc (uri "")
                                                    :scheme (if (= "https" (:scheme (uri SERVER_URL))) "wss" "ws")
                                                    :host  (:host (uri SERVER_URL))
                                                    :port  (:port (uri SERVER_URL))
                                                    :path  "/ws/")))
                            (peer-connect))} "Connect"]
       [Button {:on-click (fn [_e]
                            (send-offer))} "Send offer"]

       ])}))

(defn init []
  (rd/render
   [App]
   (js/document.getElementById "root")))
