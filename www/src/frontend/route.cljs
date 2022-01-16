(ns frontend.route
  (:require [reagent.core :as r]))

(defonce current-route (r/atom {:route-name nil
                                :params     nil
                                :query      nil}))

(defn route-name
  []
  (:route-name @current-route))

(defn params
  []
  (:params @current-route))

(defn query
  []
  (:query @current-route))

(defn on-navigate
  "A function which will be called on each route change."
  [route-name params query]
  (reset! current-route {:route-name (keyword route-name)
                         :params     params
                         :query      query}))

(defn redirect! [loc]
  (set! (.-location js/window) loc))

(def current-host
  (.-location js/window))
