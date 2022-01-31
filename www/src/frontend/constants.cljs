(ns frontend.constants)
;; Should reflect the constants in cli/pkg/message/message.go and cli/internal/cfg

;; message
(defonce TRTCOffer :Offer) ;; Offer
(defonce TRTCAnswer :Answer) ;; Answer
(defonce TRTCCandidate :Candidate) ;; Candidate
(defonce TTermWinsize :Winsize) ;; Candidate

;; connect
(defonce TCConnect         :Connect)
(defonce TCRequirePasscode :RequirePasscode)
(defonce TCNoPasscode      :NoPasscode)
;; message to wrap passcode
(defonce TCPasscode :Passcode)
;; connection's response
(defonce TCAuthenticated   :Authenticated)
(defonce TCUnauthenticated :Unauthenticated)
(defonce TCUnsupportedVersion :UnsupportedVersion)

(defonce TWSPing :Ping) ;; Candidate

;; Termishare config
(defonce TERMISHARE_WEBSOCKET_HOST_ID "host") ;; ID of message sent from the host
(defonce TERMISHARE_WEBRTC_DATA_CHANNEL "termishare") ;; lable name of webrtc data channel to exchange byte data
(defonce TERMISHARE_WEBRTC_CONFIG_CHANNEL "config") ;; lable name of webrtc config channel to exchange config
(defonce TERMISHARE_VERSION "0.0.4")
