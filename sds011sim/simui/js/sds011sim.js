/*
Main module
*/

'use strict';

/*
API:
/model  POST sets... initial get is needed
/status, stupid poll

*/

//Hack
function httpGetAsync(theUrl, callback){
  let xmlHttp = new XMLHttpRequest();
  xmlHttp.onreadystatechange = function() {
  if (xmlHttp.readyState == 4 && xmlHttp.status == 200){
      let resp=JSON.parse(xmlHttp.responseText)
    callback(resp);
    }
  }
  xmlHttp.open("GET", theUrl, true); // true for asynchronous
  xmlHttp.send(null);
}

function httpPostAsync(theUrl,payloadobject, callback){
  let xmlHttp = new XMLHttpRequest();
  xmlHttp.onreadystatechange = function() {
  if (xmlHttp.readyState == 4 && xmlHttp.status == 200){
      let resp=JSON.parse(xmlHttp.responseText)
    callback(resp);
    }
  }
  xmlHttp.open("POST", theUrl, true); // true for asynchronous
  xmlHttp.setRequestHeader("Content-Type", "application/json");
  xmlHttp.send(JSON.stringify(payloadobject));
}


var elementSingleSensor=document.querySelector("#singleSensor")

elementSingleSensor.addEventListener('userInput', e => {
  console.log("SENSOR SETTINGS CHANGED "+JSON.stringify(e.detail))
  httpPostAsync("/model",e.detail, updateModelOnUI)
})

function updateStatusToUI(newStatus){
  elementSingleSensor.setAttribute("status",JSON.stringify(newStatus)) //Attribute is string
}

let simSettingsUpdated=false //Hack
function updateModelOnUI(newModel){ //Read back etc
  simSettingsUpdated=true
  console.log("TODO MODEL UPDATE "+JSON.stringify(newModel))
  elementSingleSensor.setAttribute("model",JSON.stringify(newModel))
}

//Get and initialize

//TODO Websockets... too lazy for just dev tool lets poll
setInterval(function(){
  httpGetAsync("/status",updateStatusToUI)
  //FUCK DIRTY CHECK :D :D  if(!simSettingsUpdated){
    httpGetAsync("/model",updateModelOnUI)
  //}
},1000)
