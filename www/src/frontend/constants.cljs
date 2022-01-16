(ns frontend.constants)
;; Should reflect the constants in cli/pkg/message/message.go and cli/internal/cfg

;; message
(defonce TRTCWillYouMarryMe :WillYouMarryMe) ;; Offer
(defonce TRTCYes :Yes) ;; Answer
(defonce TRTCKiss :Kiss) ;; Candidate
(defonce TTermWinsize :Winsize) ;; Candidate

(defonce TWSPing :Ping) ;; Candidate


;; Termishare config
(defonce TERMISHARE_WEBSOCKET_HOST_ID "host") ;; ID of message sent from the host
(defonce TERMISHARE_WEBRTC_DATA_CHANNEL "termishare") ;; lable name of webrtc data channel to exchange byte data
(defonce TERMISHARE_WEBRTC_CONFIG_CHANNEL "config") ;; lable name of webrtc config channel to exchange config
