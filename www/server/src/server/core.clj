(ns server.core
  (:require [ring.adapter.jetty9 :refer [run-jetty send!] :as jetty]
            [compojure.core :refer [defroutes GET]]
            [ring.middleware.keyword-params :refer [wrap-keyword-params]]
            [ring.middleware.params :refer [wrap-params]]
            [ring.middleware.session :refer [wrap-session]]
            [compojure.route :as route])
  (:gen-class))

;; map of set of connections
(defonce connections (atom {}))

(defn ws-handler
  [roomID]
  {:on-connect (fn [ws]
                 (println (format "New connection at room %s (%d)" roomID (count (roomID @connections))))
                 (swap! connections update roomID #(if (some? %)
                                                     (conj % ws)
                                                     #{ws})))

   :on-error (fn [ws _e]
               (swap! connections update roomID disj ws))

   :on-close (fn [ws _status-code _reason]
               (swap! connections update roomID disj ws))

   :on-text (fn [ws text-message]
              ; broadcast this message to everyone except itself
              (doall (map #(send! % text-message) (filter #(not= ws %) (roomID @connections)))))})

(defroutes routes
  (GET "/ws/:id" [] (fn [{:keys [params] :as req}]
                      (when (jetty/ws-upgrade-request? req)
                        (jetty/ws-upgrade-response (ws-handler (keyword (:id params)))))))
  (route/not-found "Where are you going?"))

(def app
  (-> #'routes
      wrap-keyword-params
      wrap-params
      wrap-session))

(defn -main [& _args]
  (let [port (Integer/parseInt (or (System/getenv "TERMISHARE_PORT") "3000"))]
    (println "Serving at localhost:" port)
    (run-jetty app {:port port})))
