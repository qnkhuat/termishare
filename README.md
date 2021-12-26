# termishare
Terminal sharing via peer to peer connection

## How is it gonna work?
It will have 3 components:
- A cli app - That sharer can use to share a terminal session
- A webapp - is the webclient where we can use the shared terminal on
- A server - used as the signaling server to establish the per to peer connection



## flow
- User type `termishare` in terminal
- It creates a terminal session and establish a websocket connection with the signal server
- Signalling server then create a virtual room
- The termishare cli will output an unique link
- This link can be used to access the terminal via web
- When user click on the link, it establishes a websocket connection with signalin server
- The server then figure otu the required room and start to exchange signaling messages
- After the signaling process succeed, the terminal will be streamed to web via peer to peer connection

## TODO
- [x] Make roomable
- [ ] Keep websocket, peerconnection alive
- [ ] Set up Release
- [ ] Home page
- [ ] Message while connecting and disconnected
- [ ] Figureout do we have to have the exact webrtc config from both participants?

