(ns termishare.components.mui
  (:require [reagent.core :as r]
            ["@mui/material/Button" :as MuiButton]
            ["@mui/material/FormControl" :as MuiFormControl]
            ["@mui/material/InputLabel" :as MuiInputLabel]
            ["@mui/material/MenuItem" :as MuiMenuItem]
            ["@mui/material/Select" :as MuiSelect]
            ["@mui/material/Slider" :as MuiSlider]
            ["@mui/material/TextField" :as MuiTextField]
            ["@mui/icons-material/GitHub" :as MuiGitHubIcon]
            ["@mui/material/styles/ThemeProvider" :as MuiThemeProvider]))

(defn -adapt
  [component]
  (r/adapt-react-class (.-default component)))

(def Button (-adapt MuiButton))
(def FormControl (-adapt MuiFormControl))
(def InputLabel (-adapt MuiInputLabel))
(def TextField (-adapt MuiTextField))
(def MenuItem (-adapt MuiMenuItem))
(def Select (-adapt MuiSelect))
(def Slider (-adapt MuiSlider))
(def GitHubIcon (-adapt MuiGitHubIcon))
(def ThemeProvider (-adapt MuiThemeProvider))
