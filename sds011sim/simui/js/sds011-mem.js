/*
Element for sensor memory. Dirty check etc..

TODO not really need for dirty check.
Set once and data moves only to backend...
... ok this is hacking tool only and hacks are ok now :)

*/
'use strict';

const controlTemplate = document.createElement('template');
controlTemplate.innerHTML = `
<style>

</style>

<div class="sds011-mem">

<label>
  Id (in hex) <input id="inputId" pattern="[a-fA-F\d]+"></input> (hex)
</label>

<label>
  Measurement period <input id="inputPeriod" type="number" min="0" max="30"></input>
</label>

<label>
  Query mode
  <input id="inputQueryMode" type="checkbox"></input>
</label>


<fieldset>
  <label>
    Year 20* <input id="inputYear" type="number" min="0" max="99"></input>
  </label>
  <label>
    Month <input id="inputMonth" type="number" min="0" max="99"></input>
  </label>
  <label>
    Day <input id="inputDay" type="number" min="0" max="99"></input>
  </label>
</fieldset>


</div>
`
customElements.define('sds011-mem', class SDS011Mem extends HTMLElement {
  constructor() {
    super(); // always call super() first in the constructor.
    this._shadowRoot = this.attachShadow({ mode: 'open' });
    this._shadowRoot.appendChild(controlTemplate.content.cloneNode(true));

    this.inputControls={
      "id":this._shadowRoot.querySelector("#inputId"),
      "period":this._shadowRoot.querySelector("#inputPeriod"),
      "queryMode":this._shadowRoot.querySelector("#inputQueryMode"),
      "year":this._shadowRoot.querySelector("#inputYear"),
      "month":this._shadowRoot.querySelector("#inputMonth"),
      "day":this._shadowRoot.querySelector("#inputDay")
    }
  }

  jsonvalues(v){
    console.log("TODO ASETA "+JSON.stringify(v))
  }

  get data(){
    return {
      id:parseInt(this.inputControls.id.value,16),
      period:parseInt(this.inputControls.period.value),
      queryMode:this.inputControls.queryMode.checked,
      year:parseInt(this.inputControls.year.value),
      month:parseInt(this.inputControls.month.value),
      day:parseInt(this.inputControls.day.value)
    }
  }

  attributeChangedCallback(attrName, oldVal, newVal) {
    console.log("attribute changed callback  attrName="+attrName+" oldVal="+oldVal+" newVal="+newVal)
    let v=JSON.parse(newVal)
    this.inputControls.id.value=v.id.toString(16).toUpperCase()
    this.inputControls.period.value=v.period,

    this.inputControls.queryMode.checked=v.queryMode
    this.inputControls.year.value=v.year
    this.inputControls.month.value=v.month
    this.inputControls.day.value=v.day

    for (let name in this.inputControls){
      this.inputControls[name].addEventListener('input', e => {
        let r={}
        for (var name in this.inputControls){
          r[name]=parseInt(this.inputControls[name].value)
        }
        r.id=parseInt(this.inputControls.id.value,16)
        console.log("asetukset "+JSON.stringify(r))
        this.dispatchEvent(new CustomEvent('userInput', { detail:r }));
      });
    }
  }

  static get observedAttributes() {
    return ["mem"];
  }
})
