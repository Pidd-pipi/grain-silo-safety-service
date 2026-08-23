const silosNode=document.querySelector('#silos'),statusNode=document.querySelector('#status');
const inspectionsNode=document.querySelector('#inspections'),ordersNode=document.querySelector('#orders'),alertsNode=document.querySelector('#alerts');
async function getJSON(url){const response=await fetch(url);if(!response.ok){throw new Error(`${url}: ${response.status}`)}return response.json()}
async function load(){
  const data=await getJSON('/api/silos');
  silosNode.replaceChildren(...data.silos.map(silo=>{
    const item=document.createElement('li');
    item.textContent=`${silo.name} - ${silo.grain} - ${silo.safetyState}`;
    if(silo.safetyState!=='clear'&&!silo.inspected){
      const button=document.createElement('button');button.textContent='Record check';
      button.onclick=async()=>{await fetch(`/api/silos/${silo.id}/inspect`,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({finding:'Visual check complete'})});load()};
      item.append(' ',button)
    }
    return item
  }));
  statusNode.textContent=`${data.silos.length} silos under review`;
  try{
    const inspections=await getJSON('/api/inspections');
    inspectionsNode.replaceChildren(...inspections.items.map(i=>{const item=document.createElement('li');item.textContent=`${i.id} ${i.siloId} ${i.finding} [${i.status}]`;return item}))
  }catch(error){inspectionsNode.textContent=error.message}
  try{
    const orders=await getJSON('/api/ops/records');
    ordersNode.replaceChildren(...orders.items.map(o=>{const item=document.createElement('li');item.textContent=`${o.id} ${o.subject} [${o.status}]`;return item}))
  }catch(error){ordersNode.textContent=error.message}
  try{
    const alerts=await getJSON('/api/alerts/events');
    alertsNode.replaceChildren(...alerts.events.map(a=>{const item=document.createElement('li');item.textContent=`${a.siloId} ${a.metric} ${a.value} [${a.level}]`;return item}))
  }catch(error){alertsNode.textContent=error.message}
}
load().catch(error=>statusNode.textContent=error.message);
