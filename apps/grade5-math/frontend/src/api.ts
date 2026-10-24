export interface Module { id:string; name:string; number:string; description:string; symbol:string }
export interface Skill { id:string; moduleId:string; name:string; topicId:string; tip:string }
export interface Question { id:string; skillId:string; kind:string; purpose:string; prompt:string; options?:{id:string;text:string}[]; unit?:string; chart?:{label:string;value:number}[]; figure?:{type:string;base:number;height:number;width?:number} }
export interface Feedback { correct:boolean; skipped:boolean; explanation:string; questionId:string }
export interface View { id:string; done:boolean; step:number; total:number; phase:string; version:string; question:Question|null; skill:Skill|null; feedback:Feedback|null; modules:string[]; completedSkill?:string }
export interface Evidence { skill:Skill; status:string; correct:number; wrong:number; skipped:number; retestCorrect:number; retestCount:number; practiced:boolean }
export interface Review { question:Question; answer:string; correct:boolean; skip:string; explanation:string }
export interface Result { id:string; created:number; skills:Evidence[]; recommendation:string; tested:number; unseen:number; review:Review[] }
export interface RouteData { nodes:{id:string;name:string;sourceName:string}[]; route:{topicId:string;prerequisites:{prerequisiteId:string;strength:string}[];unlocks:{topicId:string}[]};version:string }
export async function api<T>(path:string, body?:unknown):Promise<T> {
  let response:Response;
  try { response=await fetch(`/api${path}`,{method:body===undefined?'GET':'POST',credentials:'same-origin',headers:body===undefined?{}:{'Content-Type':'application/json'},body:body===undefined?undefined:JSON.stringify(body)}); }
  catch { throw new Error('连接暂时中断，已提交的进度会保留。恢复网络后可以重试。'); }
  const data=await response.json();
  if(!response.ok) throw new Error(data.error||'暂时没有完成，请重试。');
  return data as T;
}
