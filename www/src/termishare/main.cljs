(ns termishare.main
  (:require [reagent.dom :as rd]
            [reagent.core :as r]
            [clojure.edn :as edn]
            ["xterm" :as xterm]
            ["xterm-addon-fit" :as xterm-addon-fit]
            [termishare.components.mui :refer [Button]]))

;;; ------------------------------ Utils ------------------------------
(defonce state
  (r/atom {:ws-conn       nil
           :peer-conn     nil
           :data-channels {}
           :term          nil
           :addon-fit     nil}))

(defonce text-encoder (js/TextEncoder.))
(defonce text-decoder (js/TextDecoder. "utf-8"))

(defn send-when-connected
  "Send a message via a websocket connection, Retry if it fails"
  ([ws-conn msg]
   (send-when-connected ws-conn msg 0 100))

  ([ws-conn msg n limit]
   (if (< n limit)
     (if (= (.-readyState ws-conn) 1)
       (.send ws-conn (js/JSON.stringify (clj->js msg)))
       (js/setTimeout (fn [] (send-when-connected ws-conn msg (inc n) limit)) 10))
     (js/console.log "Drop message due reached retry limits: " (clj->js msg)))))

;;; ------------------------------ Web Socket ------------------------------
(defn websocket-onmessage
  [e]
  (let [msg  (-> e .-data js/JSON.parse)
        data (-> msg .-Data js/JSON.parse)]
    (js/console.log "Recevied a message: " (clj->js msg))
    (case (keyword (.-Type msg))
      :WillYouMarryMe
      (js/console.log "We shouldn't received this question, we should be the one who asks that")
      :Yes
      (.setRemoteDescription (:peer-conn @state) data)
      :Kiss
      (->> data
           js/RTCIceCandidate.
           (.addIceCandidate (:peer-conn @state))))))

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
                         {:Type :Kiss
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
    (js/console.log "keyword msg " (clj->js msg))
    (case (-> msg :Type keyword)
      :Winsize (when-let [ws (:Data msg)]
                 (.resize (:term @state) (:Cols ws) (:Rows ws))
                 (.fit (:addon-fit @state)))

      (js/console.log "I don't know you: " (clj->js msg)))))


(defn peer-connect
  []
  (let [conn               (js/RTCPeerConnection. ice-candidate-config)
        termishare-channel (.createDataChannel conn "termishare")
        config-channel     (.createDataChannel conn "config")]
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
               (send-when-connected (:ws-conn @state) {:Type :WillYouMarryMe
                                                       :Data (js/JSON.stringify offer)})))
      (.catch (fn [e]
                (js/console.log "Failed to send offer " e)))))


;;; ------------------------------ Component ------------------------------

(defn App
  []
  (r/create-class
   {:component-did-mount
    (fn []
      (let [term      (xterm/Terminal. #js {:cursorBlink true
                                            :scrollback 1000
                                            :disableStdin false})
            addon-fit (xterm-addon-fit/FitAddon.)]
        (.loadAddon term addon-fit)
        (.onData term (fn [data] (when-let [channel (-> @state :data-channels :termishare)]
                                   (.send channel (.encode text-encoder data)))))
        (.open term (js/document.getElementById "termishare"))
        (swap! state assoc :term term)
        (swap! state assoc :addon-fit addon-fit)))

    :reagent-render
    (fn []
      [:<>
       [:div {:id "termishare"}]
       [Button {:on-click (fn [_e]
                            (ws-connect "ws://localhost:3000/ws")
                            (peer-connect))}
        "Connect"]

       [Button {:on-click (fn [_e] (send-offer))}
        "Send offer"]

       [Button {:on-click (fn [_e] (js/console.log (-> @state clj->js)))}
        "Print states"]])}))

(defn init []
  (rd/render
   [App]
   (js/document.getElementById "root")))
