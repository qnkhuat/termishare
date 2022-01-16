(ns termishare.pages.roomID
  (:require [reagent.core :as r]
            [termishare.constants :as const]
            [termishare.route :as route]
            [lambdaisland.uri :refer [uri]]
            ["xterm" :as xterm]))

;;; ------------------------------ Utils ------------------------------
(defonce state
  (r/atom {:ws-conn   nil
           :peer-conn nil
           :term      nil}))

;; msg queue to stack messages when web-socket is not connected
(defonce msg-queue (atom []))
(defonce connection-id (str (random-uuid)))
(defonce text-encoder (js/TextEncoder.))
(defonce text-decoder (js/TextDecoder. "utf-8"))
(defonce terminal-id "terminal")

(defn websocket-send-msg
  [msg]
  (.send (:ws-conn @state) (-> msg
                               (assoc
                                 :From connection-id
                                 :To   const/TERMISHARE_WEBSOCKET_HOST_ID)
                               clj->js
                               js/JSON.stringify)))

(defn send-when-connected
  "Send a message via a websocket connection, Add to a queue if it's not connected
  The msg in queue will be sent when the socket is open"
  [ws-conn msg]
  (if (and ws-conn (= (.-readyState ws-conn) 1))
    (websocket-send-msg msg)
    (swap! msg-queue conj msg)))

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
        xterm-size     (element-size (-> (js/document.getElementById terminal-id) (.querySelector ".xterm-screen")))
        new-hfont-mulp (* (/ cur-cols new-cols) (/ (:width target-size) (:width xterm-size)))
        new-vfont-mulp (* (/ cur-rows new-rows) (/ (:height target-size) (:height xterm-size)))]
    (if (> new-hfont-mulp new-vfont-mulp)
      (int (Math/floor (* cur-font-size new-vfont-mulp)))
      (int (Math/floor (* cur-font-size new-hfont-mulp))))))

;;; ------------------------------ Web Socket ------------------------------
(defn websocket-onmessage
  [e]
  (let [msg  (-> e .-data js/JSON.parse)]
    ;; only handle messages that are sent by the host to us
    (when (and (= connection-id (.-To msg))
               (= const/TERMISHARE_WEBSOCKET_HOST_ID (.-From msg)))

      (condp = (keyword (.-Type msg))

        const/TRTCWillYouMarryMe
        (js/console.log "We shouldn't received this question, we should be the one who asks that")

        const/TRTCYes
        (.setRemoteDescription (:peer-conn @state) (-> msg .-Data js/JSON.parse))

        const/TRTCKiss
        (->> (-> msg .-Data js/JSON.parse)
             js/RTCIceCandidate.
             (.addIceCandidate (:peer-conn @state)))

        const/TWSPing
        nil ; just skip it

        :else
        (js/console.error "Unhandeled message type: " (.-Type msg))))))

(defn websocket-onclose
  [e]
  (js/console.log "Websocket closed!: " e)
  (swap! state assoc :ws-conn nil))

(defn ws-connect
  "Connect to websocket server"
  [url]
  (js/console.log "Connecting to: " url)
  (when-not (:ws-conn @state)
    (let [conn (js/WebSocket. url)]
      (set! (.-onopen conn) (fn [_e]
                              (js/console.log "Websocket connected")
                              ;; Send all messages has been queued when it's not connected
                              (doall (map (fn [msg] (websocket-send-msg msg))
                                          @msg-queue))
                              (reset! msg-queue [])))
      (set! (.-onmessage conn) websocket-onmessage)
      (set! (.-onclose conn) websocket-onclose)
      (set! (.-onerror conn) websocket-onclose)
      (swap! state assoc :ws-conn conn))))

;;; ------------------------------ WebRTC ------------------------------
(def ice-candidate-config (clj->js {:iceServers [{:urls ["stun:stun.l.google.com:19302"]}
                                                 {:urls "turn:104.237.1.191:3478"
                                                  :username "termishare"
                                                  :credential "termishareisfun"}]}))


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
    (set! (.-onopen channel) (fn [] (js/console.log "Channel " (.-label channel) " opened")))))

(defn rtc-on-termishare-channel
  [e]
  (let [data (-> e .-data js/Uint8Array.)]
    (.writeUtf8 (:term @state) data)))

(defn resize
  [ws]
  (let [term-size (element-size (js/document.getElementById terminal-id))
        term      (:term @state)]
    ;; TODO this will not give a perfect sizing at first because
    ;; intially the terminal have a 80,24 size, and resizing from that will not be optimal
    (.setOption term "fontSize" (guess-new-font-size (:Cols ws) (:Rows ws)
                                                     term-size))
    (.resize term (:Cols ws) (:Rows ws))))

(defn rtc-on-config-channel
  [e]
  (let [msg (->> e .-data (.decode text-decoder) js/JSON.parse)
        msg (js->clj msg :keywordize-keys true)]
    (js/console.log "Got a message from config channel: " (clj->js msg))

    (condp = (-> msg :Type keyword)
      const/TTermWinsize
      (resize (:Data msg))
      (js/console.log "I don't know you: " (clj->js msg)))))

(defn peer-connect
  []
  (let [conn               (js/RTCPeerConnection. ice-candidate-config)
        termishare-channel (.createDataChannel conn (str const/TERMISHARE_WEBRTC_DATA_CHANNEL))
        config-channel     (.createDataChannel conn (str const/TERMISHARE_WEBRTC_CONFIG_CHANNEL))]
    (set! (.-onconnectionstatechange conn) (fn [e] (js/console.log "Peer connection state change: " (.. e -target -connectionState))))
    (set! (.-onicecandidate conn) rtc-onicecandidate)
    (set! (.-ondatachannel conn) rtc-ondatachannel)
    (set! (.-binaryType termishare-channel) "arraybuffer")
    (set! (.-binaryType config-channel) "arraybuffer")
    (set! (.-onmessage termishare-channel) rtc-on-termishare-channel)
    (set! (.-onmessage config-channel) rtc-on-config-channel)
    ;; Take user input and send to the host
    (.onData (:term @state)
             (fn [data] (.send termishare-channel (.encode text-encoder data))))
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

(defn connect
  []
  (ws-connect (str (assoc (uri "")
                          :scheme (if (= "https" (:scheme (uri route/current-host))) "wss" "ws")
                          :host   (:host (uri route/current-host))
                          :port   (:port (uri route/current-host))
                          :path   (str "/ws/" (:roomID (route/params))))))
  (peer-connect)
  (send-offer))


;;; ------------------------------ Component ------------------------------
(defn roomID
  []
  (r/create-class
    {:component-did-mount
     (fn []
       (let [term      (xterm/Terminal. #js {:cursorBlink  true
                                             :disableStdin false})]
         (.open term (js/document.getElementById terminal-id))
         (set! (.-onresize js/window) (fn [_e]
                                        (when-let [term (:term @state)]
                                          (resize {:Cols (.-cols term)
                                                   :Rows (.-rows term)}))))
         (swap! state assoc :term term)
         (connect)))
     :reagent-render
     (fn []
       [:<>
        [:div {:id terminal-id :class "w-screen h-screen fixed top-0 left-0 bg-black"}]])}))
