(ns dev
  (:require
    [ring.adapter.jetty9 :refer [run-jetty] :as jetty]
    [ring.middleware.reload :refer [wrap-reload]]
    [server.core :refer [app]]))

(println "Welcome to Termishare dev")

(defonce ^:private instance* (atom nil))

(defn start! []
  (println "Serving at localhost: 3000" )
  (reset! instance* (run-jetty (wrap-reload #'app) {:port 3000
                                                    :join? false})))

(defn stop! []
  (when @instance*
    (println "Stopping")
    (.stop @instance*)
    (reset! instance* nil)))

(defn restart! []
  (stop!)
  (start!))
