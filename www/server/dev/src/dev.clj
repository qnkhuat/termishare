(ns dev
  (:require
    [ring.adapter.jetty9 :refer [run-jetty] :as jetty]
    [server.core :refer [app]]))

(println "Welcome to Termishare dev")

(defonce ^:private instance* (atom nil))

(defn instance []
  @instance*)

(defn start! []
  (reset! instance* (run-jetty app {:port 3000})))

(defn stop! []
  (when (instance)
    (.stop (instance))))


