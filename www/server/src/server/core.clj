(ns server.core
  (:require [ring.adapter.jetty9 :refer [run-jetty send!] :as jetty]
            [compojure.core :refer [defroutes GET]]
            [ring.middleware.keyword-params :refer [wrap-keyword-params]]
            [ring.middleware.params :refer [wrap-params]]
            [ring.middleware.session :refer [wrap-session]]
            [compojure.route :as route])
  (:gen-class))

(defonce connections (atom #{}))

(def ws-handler {:on-connect (fn [ws]
                               (println "New connection : " ws)
                               (swap! connections conj ws))

                 :on-error (fn [ws _e]
                             (swap! connections disj ws))

                 :on-close (fn [ws _status-code _reason]
                             (swap! connections disj ws))

                 :on-text (fn [ws text-message]
                            ; broadcast this message to everyone except itself
                            (println "Broadcasted to " (-> @connections count dec))
                            (doall (map #(send! % text-message) (filter #(not= ws %) @connections))))})

(defroutes routes
  (GET "/ws/:id" [] (fn [req]
                      (println "id: " (:id (:params req)))
                      (if (jetty/ws-upgrade-request? req)
                        (jetty/ws-upgrade-response ws-handler))))
  (route/not-found "Where are you going?"))

(def app
  (-> #'routes
      wrap-keyword-params
      wrap-params
      wrap-session))

;(defn app [req]
;  (if (jetty/ws-upgrade-request? req)
;    (jetty/ws-upgrade-response ws-handler)))


(defn -main [& _args]
  (let [port (Integer/parseInt (or (System/getenv "TERMISHARE_PORT") "3000"))]
    (println "Serving at localhost:" port)
    (run-jetty app {:port port})))
