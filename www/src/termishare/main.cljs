(ns termishare.main
  (:require [bide.core :as bide]
            [reagent.dom :as rd]
            [termishare.route :as route]
            [termishare.pages.roomID :refer [roomID]]))

(defonce router
  (bide/router [["/" :home]
                ["/:roomID" :roomID]]))

(defn current-page
  []
  (js/console.log "route-name: " (route/route-name))
  (case (route/route-name)
    :home   [:h3 "Home"]
    :roomID [roomID]
    [:h3 "404"]))

(defn route-init
  []
  (bide/start! router {:default     :home
                       :on-navigate route/on-navigate
                       :html5?      true}))

(defn init
  []
  (route-init)
  (rd/render
   [:div (current-page)]
   (js/document.getElementById "root")))
