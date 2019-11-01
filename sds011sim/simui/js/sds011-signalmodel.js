
/*
sds011 signal model.  Just set level, add noise and maybe sine component with long period ~hours?
*/

'use strict';

const controlTemplate = document.createElement('template');
controlTemplate.innerHTML = `
<style>

</style>

<div class="sds011-signalmodel">
  <label>
    Noise  <input id="inputNoise" type="range" value="0" min="0" max="999.9" step="0.1"></input> <span id="lblNoise"></span>
  </label>
  <label>
    Offset  <input id="inputOffset" type="range" value="0" min="0" max="999.9" step="0.1"></input> <span id="lblOffset"></span>
  </label>
  <label>
    Period   <input id="inputPeriod" type="range" value="0" min="0" max="3600000" step="1"></input> <span id="lblPeriod"></span>
  </label>
  <label>
    Phase <input id="inputPhase" type="range" value="0" min="0" max="3600000" step="1"></input> <span id="lblPhase"></span>
  </label>
  <label>
    Amplitude <input id="inputAmplitude" type="range" value="0" min="0" max="999.9" step="0.1"></input> <span id="lblAmplitude"></span>
  </label>
</div>
`

customElements.define('sds011-signalmodel', class SDS011SignalModel extends HTMLElement {
  constructor() {
    super(); // always call super() first in the constructor.

    this._shadowRoot = this.attachShadow({ mode: 'open' });
    this._shadowRoot.appendChild(controlTemplate.content.cloneNode(true));
    this.inputControls={
      "noise":this._shadowRoot.querySelector("#inputNoise"),
      "offset":this._shadowRoot.querySelector("#inputOffset"),
      "period":this._shadowRoot.querySelector("#inputPeriod"),
      "phase":this._shadowRoot.querySelector("#inputPhase"),
      "amplitude":this._shadowRoot.querySelector("#inputAmplitude")
    }

    this.labelControls={
      "noise":this._shadowRoot.querySelector("#lblNoise"),
      "offset":this._shadowRoot.querySelector("#lblOffset"),
      "period":this._shadowRoot.querySelector("#lblPeriod"),
      "phase":this._shadowRoot.querySelector("#lblPhase"),
      "amplitude":this._shadowRoot.querySelector("#lblAmplitude")
    }


    for (let name in this.inputControls){
      this.inputControls[name].addEventListener('input', e => {
        this.controlsChanged();
        let r={}
        for (var name in this.inputControls){
          r[name]=parseFloat(this.inputControls[name].value)
        }

        this.dispatchEvent(new CustomEvent('userInput', { detail:r }));
      });
    }
  }

  attributeChangedCallback(attrName, oldVal, newVal) {
    console.log("SIGNAL GEN attribute changed callback  attrName="+attrName+" oldVal="+oldVal+" newVal="+newVal)

    let obj=JSON.parse(newVal)

    for(let name in this.inputControls){
      if (obj[name]!=undefined){
        this.inputControls[name].value=obj[name]
        this.labelControls[name].innerHTML=obj[name]
      }
    }
  }

  controlsChanged(){
    let r={}
    for(let name in this.inputControls){
      r[name]=this.inputControls[name].value
      this.labelControls[name].innerHTML=r[name]
    }
  }

  static get observedAttributes() {
    return ["mem"];
  }

})
