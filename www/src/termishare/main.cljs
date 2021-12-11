(ns termishare.main
  (:require [reagent.dom :as rd]
            [reagent.core :as r]
            [clojure.edn :as edn]
            ["xterm" :as xterm]
            [termishare.components.mui :refer [Button]]))

;;; ------------------------------ Utils ------------------------------
(defn send-when-connected
  "Send a message via a websocket connection, Retry if it fails"
  ([ws-conn msg]
   (send-when-connected ws-conn msg 0 100))

  ([ws-conn msg n limit]
   (if (< n limit)
     (if (= (.-readyState ws-conn) 1)
       (do
        (js/console.log "Sending out:" (clj->js msg))
        (.send ws-conn (js/JSON.stringify (clj->js msg))))
       (js/setTimeout (fn [] (send-when-connected ws-conn msg (inc n) limit)) 10))
     (js/console.log "Drop message due reached retry limits: " (clj->js msg)))))

(defonce state
  (r/atom {:ws-conn      nil
           :peer-conn    nil
           :data-channel nil}))

;;; ------------------------------ Web Socket ------------------------------
(defn websocket-onmessage
  [e]
  (let [msg  (-> e .-data js/JSON.parse)
        data (-> msg .-Data js/JSON.parse)]
    (js/console.log "Recevied a message: " (clj->js msg))
    (case (keyword (.-Type msg))
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

(defn channel-onmessage
  [e]
  (js/console.log "Channel received a message: " (-> e .-data clj->js)))

(defn peer-connect
  []
  (let [conn    (js/RTCPeerConnection. ice-candidate-config)
        channel (.createDataChannel conn "termishare")]
    (set! (.-onconnectionstatechange conn) (fn [e] (js/console.log "Peer connection state change: " (clj->js e))))
    (set! (.-onicecandidate conn) rtc-onicecandidate)
    (set! (.-ondatachannel conn) rtc-ondatachannel)
    (set! (.-onmessage channel) channel-onmessage)
    (swap! state assoc :data-channel channel)
    (swap! state assoc :peer-conn conn)))

(defn add-tracks
  [stream]
  (doseq [track (.getTracks stream)]
    (js/console.log "adding track: " track)
    (.addTrack (:peer-conn @state) track)))

(defn send-offer
  []
  (-> js/navigator
      .-mediaDevices
      (.getUserMedia #js {:video true :audio false})
      (.then (fn [stream]
               (add-tracks stream)))
      (.then (fn []
               (-> (:peer-conn @state)
                   .createOffer
                   (.then (fn [offer]
                            (.setLocalDescription (:peer-conn @state) offer)
                            (send-when-connected (:ws-conn @state) {:Type :WillYouMarryMe
                                                                    :Data (js/JSON.stringify offer)})))
                   (.catch (fn [e]
                             (js/console.log "Failed to send offer " e))))))))

;;; ------------------------------ Component ------------------------------
(defn App
  []
  (r/create-class
   {:component-did-mount
    (fn []
     ; (let [term (xterm/Terminal.)]
     ;   (.open term (js/document.getElementById "termishare"))
     ;   (.write term "Hello"))
      )

    :reagent-render
    (fn []
      [:<>
       ;[:div {:id "termishare"}]
       [:h1 {:class "font-bold text-blue-400"} "Hello boissss"]
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
