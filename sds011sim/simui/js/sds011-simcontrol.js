/*
sds011-simcontrol

This controls SDS011 simulator instance

*/

'use strict';

const controlTemplate = document.createElement('template');
controlTemplate.innerHTML = `
<style>

</style>

<div class="sds011-simcontrol">

  <label>
    PowerOn  <input id="powerOn" type="checkbox"></input>
  </label>

  <ul>
    <li>Working: <span id="working"> </span> </li>
    <li>MeasurementCounter:<span id="measurementCounter"> </span> </li>
    <li>RX Packet Counter:<span id="rxPacketCounter"> </span> </li>
    <li>TX Packet Counter:<span id="txPacketCounter"> </span> </li>
    <li>SmallRegNow:<span id="smallRegNow"> </span> </li>
    <li>LargeRegNow:<span id="largeRegNow"> </span> </li>
    <li>Burn Event Counter:<span id="burnEventCounter"> </span> </li>
  </ul>

  <h2>Non-volatile</h2>
    <sds011-mem id="memElement">{"id":1234,"period":0,"queryMode":false,"year":19,"month":10,"day":3} </sds011-mem>

  <h2>Connectivity</h2>
    <sds011-connectivitymodel id="connectivityElement">
      {"rxConnected":true,"txConnected":true,"shortCircuit":false,"directionChangeNull":false,"incompletePackages":false,"invalidCRC":false,"idleCharacters":false}
    </sds011-connectivitymodel>

  <h2>Signal generator</h2>
    <sds011-signalmodel id="smallParticleSignal">
    </sds011-signalmodel>

  <h2>Large</h2>
    <sds011-signalmodel id="largeParticleSignal">
    </sds011-signalmodel>

</div>
`

var modelData={}
customElements.define('sds011-simcontrol', class SDS011SimControl extends HTMLElement {

  constructor() {
    super(); // always call super() first in the constructor.
    //let elem=this._shadowRoot.querySelector(".sds011-mem")

    this._shadowRoot = this.attachShadow({ mode: 'open' });
    this._shadowRoot.appendChild(controlTemplate.content.cloneNode(true));
    this.elemMap={
      "powerOn":this._shadowRoot.querySelector("#powerOn"),
      "working":this._shadowRoot.querySelector("#working"),
      "measurementCounter":this._shadowRoot.querySelector("#measurementCounter"),
      "rxPacketCounter":this._shadowRoot.querySelector("#rxPacketCounter"),
      "txPacketCounter":this._shadowRoot.querySelector("#txPacketCounter"),
      "smallRegNow":this._shadowRoot.querySelector("#smallRegNow"),
      "largeRegNow":this._shadowRoot.querySelector("#largeRegNow"),
      "burnEventCounter":this._shadowRoot.querySelector("#burnEventCounter"),

      "memElement":this._shadowRoot.querySelector("#memElement"),
      "connectivityElement":this._shadowRoot.querySelector("#connectivityElement"),

      "smallParticleSignal":this._shadowRoot.querySelector("#smallParticleSignal"),
      "largeParticleSignal":this._shadowRoot.querySelector("#largeParticleSignal")
    }

    this.elemMap.powerOn.addEventListener('change', e => {
      //console.log("POWERIKYTKIN "+this.elemMap.powerOn.checked)
      modelData.powerOn=this.elemMap.powerOn.checked
      //console.log("SMALL gen changed "+JSON.stringify(e.detail))
      //modelData.smallParticles=e.detail
      this.dispatchEvent(new CustomEvent('userInput', { detail:modelData }));
    })

    this.elemMap.smallParticleSignal.addEventListener('userInput', e => {
      console.log("SMALL gen changed "+JSON.stringify(e.detail))
      modelData.smallParticles=e.detail
      this.dispatchEvent(new CustomEvent('userInput', { detail:modelData }));
    })

    this.elemMap.largeParticleSignal.addEventListener('userInput', e => {
      console.log("LARGE gen changed "+JSON.stringify(e.detail))
      modelData.largeParticles=e.detail
      this.dispatchEvent(new CustomEvent('userInput', { detail:modelData }));
    })

    this.elemMap.memElement.addEventListener('userInput', e => {
      console.log("mem changed "+JSON.stringify(e.detail))
      modelData.sensorMem=e.detail
      this.dispatchEvent(new CustomEvent('userInput', { detail:modelData }));
    })

    this.elemMap.connectivityElement.addEventListener('userInput', e => {
      console.log("connectivity changed "+JSON.stringify(e.detail))
      modelData.connectivity=e.detail
      this.dispatchEvent(new CustomEvent('userInput', { detail:modelData }));
    })
  }

  attributeChangedCallback(attrName, oldVal, newVal) {
    let obj=JSON.parse(newVal) //TODO TRY CATCH

    if (attrName=="status"){
      //Hack :D
      this.elemMap.working.innerHTML=obj.working
      this.elemMap.measurementCounter.innerHTML=" "+obj.measurementCounter
      this.elemMap.rxPacketCounter.innerHTML=" "+obj.rxPacketCounter
      this.elemMap.txPacketCounter.innerHTML=" "+obj.txPacketCounter
      this.elemMap.smallRegNow.innerHTML=" "+obj.smallRegNow
      this.elemMap.largeRegNow.innerHTML=" "+obj.largeRegNow
      this.elemMap.burnEventCounter.innerHTML=" "+obj.burnEventCounter
    }
    if (attrName=="model"){
      console.log("TODO update model controls "+newVal)
      console.log("SET mem attrib "+JSON.stringify(obj.sensorMem))
      this.elemMap.memElement.setAttribute("mem",JSON.stringify(obj.sensorMem))

      this.elemMap.powerOn.checked=obj.powerOn
      this.elemMap.smallParticleSignal.setAttribute("mem",JSON.stringify(obj.smallParticles))
      this.elemMap.largeParticleSignal.setAttribute("mem",JSON.stringify(obj.largeParticles))
      this.elemMap.connectivityElement.setAttribute("mem",JSON.stringify(obj.connectivity))

      modelData=obj
    }
  }

  static get observedAttributes() {
    //Readonly mode changes for view only
    return ["status","model"];
  }
})
