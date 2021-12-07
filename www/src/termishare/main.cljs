(ns termishare.main
  (:require [reagent.dom :as rd]
            [reagent.core :as r]
            [clojure.edn :as edn]
            [termishare.components.mui :refer [Button]]))
(defn App
  []
  [:<>
   [:h1 "Hello bois"]
   [Button "Click me babe"]])

(defn init []
  (rd/render
   [App]
   (js/document.getElementById "root")))
