(ns frontend.main
  (:require [bide.core :as bide]
            [reagent.dom :as rd]
            [frontend.route :as route]
            [frontend.components.mui :refer [ThemeProvider]]
            ["@mui/material/styles" :refer [createTheme]]
            [frontend.pages.roomID :refer [roomID]]
            [frontend.pages.index :refer [index]]))

(defonce router
  (bide/router [["/" :home]
                ["/:roomID" :roomID]]))

(defn current-page
  []
  (case (route/route-name)
    :home   [index]
    :roomID [roomID]
    [:h3 "404"]))

(def theme (createTheme (clj->js {:palette {:mode "dark"}})))

(defn route-init
  []
  (bide/start! router {:default     :home
                       :on-navigate route/on-navigate
                       :html5?      true}))

(defn init
  []
  (route-init)
  (rd/render
    [ThemeProvider {:theme theme}
     [:<> (current-page)]]
    (js/document.getElementById "root")))
