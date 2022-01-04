(ns server.core
  (:require [ring.adapter.jetty9 :refer [run-jetty send!]]
            [compojure.core :refer [defroutes]]
            [ring.middleware.keyword-params :refer [wrap-keyword-params]]
            [ring.middleware.params :refer [wrap-params]]
            [ring.middleware.session :refer [wrap-session]]
            [compojure.route :as route])
  (:gen-class))

; TODO: use https://github.com/jarohen/chord instead of jetty 9 websocket

(defonce connections (atom #{}))

(defroutes routes
  (route/not-found "Where are you going?"))

(def ws-handler {:on-connect (fn [ws]
                               (println "New connection")
                               (swap! connections conj ws))

                 :on-error (fn [ws _e]
                             (swap! connections disj ws))

                 :on-close (fn [ws _status-code _reason]
                             (swap! connections disj ws))

                 :on-text (fn [ws text-message]
                            ; broadcast this message to everyone except itself
                            (println "Broadcasted to " (-> @connections count dec))
                            (doall (map #(send! % text-message) (filter #(not= ws %) @connections))))})

(def websocket-routes {"/ws" ws-handler})

(def app
  (-> #'routes
      wrap-keyword-params
      wrap-params
      wrap-session))

(defn -main [& _args]
  (let [port (Integer/parseInt (or (System/getenv "COLAB_PORT") "3000"))]
    (printf "Serving at ::%d\n" port)(flush)
    (run-jetty app {:port port
                    :websockets websocket-routes})))
