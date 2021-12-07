const conn = new WebSocket("ws://localhost:3000/ws");

const pc = new RTCPeerConnection({
  iceServers: [
    {
      urls: 'stun:stun.l.google.com:19302'
    }
  ]
})

let sendChannel;

const handleMessage = (ev) => {
  const msg = JSON.parse(ev.data);
  console.log("Received a message:", msg);

  switch (msg.Type) {
    case "WillYouMarryMe":
      console.log("We shouldn't be the one who get asked");
      break;

    case "Yes":
      pc.setRemoteDescription(JSON.parse(msg.Data));
      break;

    case "Kiss":
      const candidate = new RTCIceCandidate(JSON.parse(msg.Data));
      pc.addIceCandidate(candidate);
      break;
  }
}

pc.onicecandidate = (e) => {
  if (e.candidate) {
    conn.send(JSON.stringify({
      Type: "Kiss",
      Data: JSON.stringify(e.candidate.toJSON())}))
  }
}

conn.onmessage = handleMessage;

conn.onopen = () => {
  console.log("Websocket connected!");
}

const clickToSend = async () => {
  sendChannel = pc.createDataChannel("chat");
  sendChannel.onopen = () => {
    const readyState = sendChannel.readyState;
    console.log("Send channel state is: ", readyState);
  }

  // get media tracks
  const stream = await navigator.mediaDevices.getUserMedia({audio:true});
  const tracks = stream.getTracks()
  tracks.forEach((track) => pc.addTrack(track));

  // send offer
  const offer = await pc.createOffer();
  pc.setLocalDescription(offer);
  const msg = JSON.stringify({Type: "WillYouMarryMe", Data: JSON.stringify(offer)})
  conn.send(msg);
}


const clickToChat = async () => {
  sendChannel.send("ALOOOOOOOOO");
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


