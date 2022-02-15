;; It's intentional to make the server to be as dump as possible
;; All it does it broadcast messages from one client to others,
;; it doesn't know who is client, who is server.
;; That why when sending message we need to explicitly
;; say where message come from and who it is sent to
(ns server.core
  (:require [ring.adapter.jetty9 :refer [run-jetty send!] :as jetty]
            [compojure.core :refer [defroutes GET]]
            [compojure.route :as route]
            [ring.middleware.keyword-params :refer [wrap-keyword-params]]
            [ring.middleware.params :refer [wrap-params]]
            [ring.middleware.session :refer [wrap-session]]
            [ring.middleware.resource :refer [wrap-resource]]
            [ring.util.response :refer [resource-response]]
            [ring.logger :refer [wrap-log-request-start wrap-log-response]])
  (:gen-class))

(def frontend-root "frontend") ;; relative to target/classes/ on prod, or resources in development

(def log-fn (fn [{:keys [level throwable message]}]
              (println level throwable message)))

;; map of set of connections
(defonce connections (atom {}))

(defn ws-handler
  [roomID]
  {:on-connect (fn [ws]
                 (swap! connections update roomID #(if (nil? %)
                                                     #{ws}
                                                     (conj % ws))))

   :on-error (fn [ws _e]
               (swap! connections update roomID disj ws))

   :on-close (fn [ws _status-code _reason]
               (swap! connections update roomID disj ws))

   :on-text (fn [ws text-message]
              ; broadcast this message to everyone except itself
              (doall (map #(send! % text-message) (filter #(not= ws %) (roomID @connections)))))})

(defroutes routes
  (GET "/api/health" []
       (fn [_req]
         "fine"))

  (GET "/ws/:id" []
       (fn [{:keys [params] :as req}]
         (when (jetty/ws-upgrade-request? req)
           (jetty/ws-upgrade-response (ws-handler (keyword (:id params)))))))

  (GET "/" []
       (fn [_req]
         (resource-response "index.html" {:root frontend-root})))

  (GET "/:sessionId" []
       (fn [_req]
         (resource-response "index.html" {:root frontend-root})))

  (route/not-found "<h1>Page not found</h1>"))

(def app
  (-> #'routes
      wrap-keyword-params
      wrap-params
      wrap-session
      (wrap-log-request-start {:log-fn log-fn})
      (wrap-log-response {:log-fn log-fn})
      (wrap-resource frontend-root)))

(defn -main [& _args]
  (let [host (or (System/getenv "TERMISHARE_HOST") "localhost")
        port (Integer/parseInt (or (System/getenv "TERMISHARE_PORT") "3000"))]
    (println (format "Serving at %s:%s" host port))
    (run-jetty app {:port port
                    :host host})))
