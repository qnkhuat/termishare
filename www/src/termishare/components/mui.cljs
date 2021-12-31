(ns termishare.components.mui
  (:require [reagent.core :as r]
            ["@mui/material/Button" :as MuiButton]
            ["@mui/material/FormControl" :as MuiFormControl]
            ["@mui/material/InputLabel" :as MuiInputLabel]
            ["@mui/material/MenuItem" :as MuiMenuItem]
            ["@mui/material/Select" :as MuiSelect]
            ["@mui/material/Slider" :as MuiSlider]
            ["@mui/icons-material/GitHub" :as MuiGitHubIcon]))

(defn -adapt
  [component]
  (r/adapt-react-class (.-default component)))

(def Button (-adapt MuiButton))
(def FormControl (-adapt MuiFormControl))
(def InputLabel (-adapt MuiInputLabel))
(def MenuItem (-adapt MuiMenuItem))
(def Select (-adapt MuiSelect))
(def Slider (-adapt MuiSlider))
(def GitHubIcon (-adapt MuiGitHubIcon))
