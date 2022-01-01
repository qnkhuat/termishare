(ns termishare.pages.index
  (:require [termishare.components.mui :refer [GitHubIcon TextField Button]]))

(defn index
  []
  [:div {:class "bg-black w-screen h-screen"}
   [:div {:class "w-72 sm:w-96 h-screen flex flex-col justify-center text-white m-auto"}
    [:p  {:class "mb-2 font-bold text-3xl decoration-2 decoration-lime-500 underline decoration-wavy underline-offset-4 hover:underline-offset-8"} "Termishare"]
    [:p "Peer to peer terminal sharing "]
    [:div {:class "flex mt-4 mb-4"}
     [TextField {:label "Session ID" :variant "filled" :color "success" :className "w-64 sm:w-80 mr-4"}]
     [Button {:className "border-2 border-lime-500 bg-lime-500 text-white font-bold"}"Join"]]
    [:a {:class "text-white text-center font-bold text-lg sm:text-xl mb-24
                "
         :href "https://github.com/qnkhuat/termishare"}
     [GitHubIcon {:className "animate-pulse hover:animate-bounce"}]
     ]]])
