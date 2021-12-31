(ns termishare.pages.index
  (:require [termishare.components.mui :refer [GitHubIcon]]))

(defn index
  []
  [:div {:class "w-screen h-screen flex items-center justify-center bg-black"}
   [:a {:class "text-white text-center font-bold text-xl  mb-24
               decoration-sky-500 hover:decoration-sky-200 decoration-2 underline decoration-wavy underline-offset-4"
        :href "https://github.com/qnkhuat/termishare"}
    [GitHubIcon {:class "mb-2"}]
    [:br]
    "github.com/qnkhuat/termishare"]])
