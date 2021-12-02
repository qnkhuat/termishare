const conn = new WebSocket("ws://localhost:3000/ws");

const pc = new RTCPeerConnection({
  iceServers: [
    {
      urls: 'stun:stun.l.google.com:19302'
    }
  ]
})

const handleMessage = (ev) => {
  const msg = JSON.parse(ev.data);
  console.log("Received a message:", msg);

  switch (msg.Type) {
    case "WillYouMarryMe":
      console.log("We shouldn't be the one who get asked");
      break;

    case "Yes":
      pc.setRemoteDescription(msg.Data);
      break;

    case "Kiss":
      const candidate = RTCIceCandidate(msg.Data);
      pc.addIceCandidate(candidate);
      break;
  }
}

pc.onicecandidate = (e) => {
  if (e.candidate) {
    conn.send(JSON.stringify({
      Type: "Kiss",
      Data: e.candidate.toJSON()}))
  }
}

conn.onmessage = handleMessage;

conn.onopen = () => {
  console.log("Websocket connected!");
}


const clickToSend = () => {
  console.log("Sending");

  // send offer
  const offer = pc.createOffer();
  pc.setLocalDescription(offer);
  conn.send(JSON.stringify({Type: "WillYouMarryMe", Data: offer}))

}
pc.ondatachannel = e => {
  let dc = e.channel
  log('New DataChannel ' + dc.label)
  dc.onclose = () => console.log('dc has closed')
  dc.onopen = () => console.log('dc has opened')
  dc.onmessage = e => log(`Message from DataChannel '${dc.label}' payload '${e.data}'`)
  window.sendMessage = () => {
    let message = document.getElementById('message').value
    if (message === '') {
      return alert('Message must not be empty')
    }

    dc.send(message)
  }
}


