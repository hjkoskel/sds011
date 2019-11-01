/*

*/
'use strict';

const controlTemplate = document.createElement('template');
controlTemplate.innerHTML = `
<style>

</style>
<div class="sds011-connectivitymodel">

<ul>
<li> RxConnected <input id="rxConnected" type="checkbox"></input> </li>
<li> TxConnected <input id="txConnected" type="checkbox"></input> </li>
<li> Short circuit <input id="shortCircuit" type="checkbox"></input> </li>
<li> Direction change null print <input id="directionChangeNull" type="checkbox"></input> </li>
<li> Incomplete packages <input id="incompletePackages" type="checkbox"></input> </li>
<li> Invalid CRC <input id="invalidCRC" type="checkbox"></input> </li>
<li> Idle characters <input id="idleCharacters" type="checkbox"></input> </li>
</ul>

</div>
`

customElements.define('sds011-connectivitymodel', class SDS011Mem extends HTMLElement {
  constructor() {
    super(); // always call super() first in the constructor.

    this._shadowRoot = this.attachShadow({ mode: 'open' });
    this._shadowRoot.appendChild(controlTemplate.content.cloneNode(true));

    this.inputControls={
      "rxConnected":this._shadowRoot.querySelector("#rxConnected"),
      "txConnected":this._shadowRoot.querySelector("#txConnected"),
      "shortCircuit":this._shadowRoot.querySelector("#shortCircuit"),
      "directionChangeNull":this._shadowRoot.querySelector("#directionChangeNull"),
      "incompletePackages":this._shadowRoot.querySelector("#incompletePackages"),
      "invalidCRC":this._shadowRoot.querySelector("#invalidCRC"),
      "idleCharacters":this._shadowRoot.querySelector("#idleCharacters")
    }
    for (let name in this.inputControls){
      this.inputControls[name].addEventListener('input', e => {
        let r={}
        for (var name in this.inputControls){
          r[name]=this.inputControls[name].checked
        }
        console.log("Ruksit "+JSON.stringify(r))
        this.dispatchEvent(new CustomEvent('userInput', { detail:r }));
      });
    }
  }

  attributeChangedCallback(attrName, oldVal, newVal) {
    console.log("attribute changed callback  attrName="+attrName+" oldVal="+oldVal+" newVal="+newVal)
    let v=JSON.parse(newVal)

    for(let name in this.inputControls){
      if(v[name]!=undefined){
        this.inputControls[name].checked=v[name]
      }
    }
  }

  static get observedAttributes() {
    return ["mem"];
  }

})
