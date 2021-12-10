(ns termishare.main
  (:require [reagent.dom :as rd]
            [reagent.core :as r]
            [clojure.edn :as edn]
            ["xterm" :as xterm]
            [termishare.components.mui :refer [Button]]))
(defn App
  []
  (r/create-class
   {:component-did-mount
    (fn []
      (let [term (xterm/Terminal.)]
        (.open term (js/document.getElementById "termishare"))
        (.write term "Hello")))
    :reagent-render
    (fn []
      [:<>
       [:div {:id "termishare"}]
       [:h1 "Hello boissss"]
       [Button "Click me babe"]])}))

(defn init []
  (rd/render
   [App]
   (js/document.getElementById "root")))
