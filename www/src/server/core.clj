(ns server.core
  (:require [ring.adapter.jetty9 :refer [run-jetty send!] :as jetty]
            [compojure.core :refer [defroutes GET]]
            [ring.middleware.keyword-params :refer [wrap-keyword-params]]
            [ring.middleware.params :refer [wrap-params]]
            [ring.middleware.session :refer [wrap-session]]
            [ring.middleware.file :refer [wrap-file]]
            [ring.util.response :refer [file-response]])
  (:gen-class))

(def frontend-root "target/classes/public/termishare/")

(defn this-jar
  "utility function to get the name of jar in which this function is invoked"
  [ns]
  ;; The .toURI step is vital to avoid problems with special characters,
  ;; including spaces and pluses.
  ;; Source: https://stackoverflow.com/q/320542/7012#comment18478290_320595
  (-> (or ns (class *ns*))
      .getProtectionDomain .getCodeSource .getLocation .toURI .getPath))

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
  (GET "/" []
       (fn [_req]
         (file-response "index.html" {:root frontend-root})))
  (GET "/:sessionId" []
       (fn [_req]
         (file-response "index.html" {:root frontend-root}))))


(def app
  (-> #'routes
      wrap-keyword-params
      wrap-params
      wrap-session
      (wrap-file frontend-root)))

(defn -main [& _args]
  (let [port (Integer/parseInt (or (System/getenv "TERMISHARE_PORT") "3000"))]
    (println "Serving at localhost:" port)
    (println "This jar: " (this-jar server.core))
    (run-jetty app {:port port})))