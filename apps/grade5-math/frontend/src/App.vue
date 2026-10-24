<script setup lang="ts">
import { newRequestKey } from './request-key';
import { computed, onMounted, onUnmounted, ref } from 'vue';
import { api, type Module, type Skill, type View, type Result, type RouteData, type Evidence, type Feedback } from './api';

const modules=ref<Module[]>([]), skills=ref<Skill[]>([]), selected=ref<string[]>([]);
const page=ref<'home'|'question'|'result'>('home');
const session=ref<View|null>(null), latest=ref<View|null>(null), result=ref<Result|null>(null), route=ref<RouteData|null>(null);
const selectedSkill=ref(''), answer=ref(''), error=ref(''), busy=ref(false), loading=ref(true), pendingFeedback=ref<Feedback|null>(null);
const online=ref(navigator.onLine), sourceDialog=ref<HTMLDialogElement|null>(null), questionHeading=ref<HTMLElement|null>(null);
const startKey=ref(newRequestKey());
const labels:Record<string,string>={untested:'尚未检测',outside:'本次未选',not_learned:'还没学过',confirm:'需要再确认',uncertain:'需要再确认',observed:'完成一道题',practice:'建议巩固',smooth:'本次初步顺畅'};
const moduleName=(id:string)=>modules.value.find(m=>m.id===id)?.name||'';
const activeEvidence=computed(()=>result.value?.skills.find(e=>e.skill.id===selectedSkill.value));
const passed=computed(()=>result.value?.skills.filter(e=>e.retestCount===2&&e.retestCorrect===2).length||0);
const q=computed(()=>session.value?.question);
const phaseLabel=computed(()=>pendingFeedback.value?'看一看思路':session.value?.phase==='practice'?'练一练':session.value?.phase==='retest'?'换题复测':session.value?.phase==='confirm'?'再确认一下':'轻松小体检');
const canSubmit=computed(()=>answer.value.trim()!==''&&!busy.value);
const progress=computed(()=>session.value?session.value.step/session.value.total*100:0);
const routeNodes=computed(()=>{
  if(!route.value||!activeEvidence.value)return [];
  const name=(id:string)=>route.value?.nodes.find(n=>n.id===id)?.name||'相关知识';
  return [
    ...route.value.route.prerequisites.slice(0,2).map(e=>({id:e.prerequisiteId,name:name(e.prerequisiteId),tag:e.strength==='hard'?'前置基础':'辅助理解',current:false})),
    {id:route.value.route.topicId,name:activeEvidence.value.skill.name,tag:'当前关注',current:true},
    ...route.value.route.unlocks.slice(0,1).map(e=>({id:e.topicId,name:name(e.topicId),tag:'以后可能用到',current:false})),
  ];
});
function evidenceText(e:Evidence){
  if(e.status==='outside')return '可以在下一次体检时选择这个模块';
  if(e.status==='not_learned')return '先留着，等学过以后再来试试';
  if(e.correct+e.wrong===0)return e.skipped?'还没有足够的独立作答证据':'本次还没有检查这个知识点';
  return `${e.correct+e.wrong} 道独立作答 · ${e.correct} 道正确${e.skipped?` · ${e.skipped} 道跳过`:''}`;
}
function toggle(id:string){selected.value=selected.value.includes(id)?selected.value.filter(x=>x!==id):[...selected.value,id];}
async function task(fn:()=>Promise<void>){if(busy.value)return;busy.value=true;error.value='';try{await fn();}catch(e){error.value=e instanceof Error?e.message:'暂时无法完成，请重试。';}finally{busy.value=false;}}
async function chooseRoute(id:string){
  selectedSkill.value=id;route.value=null;
  if(!session.value)return;
  const data=await api<RouteData>(`/sessions/${session.value.id}/route/${id}`);
  if(selectedSkill.value===id)route.value=data;
}
async function showResult(){
  if(!session.value)return;
  result.value=await api<Result>(`/sessions/${session.value.id}/result`);page.value='result';pendingFeedback.value=null;
  const id=session.value.completedSkill||result.value.recommendation||result.value.skills.find(e=>e.status!=='outside')?.skill.id;
  if(id)await chooseRoute(id);
  window.scrollTo({top:0,behavior:'smooth'});
}
async function applyView(view:View, feedback=true){
  session.value=view;latest.value=view;answer.value='';
  if(view.done){await showResult();return;}
  page.value='question';pendingFeedback.value=feedback?view.feedback:null;
  window.scrollTo({top:0,behavior:'smooth'});
}
async function start(){await task(async()=>{
  if(!selected.value.length)throw new Error('选一个学过的模块，就可以开始啦。');
  const view=await api<View>('/sessions',{modules:selected.value,key:startKey.value});
  startKey.value=newRequestKey();await applyView(view);
});}
async function resume(){await task(async()=>{if(latest.value)await applyView(await api<View>(`/sessions/${latest.value.id}`));});}
async function submit(skip=''){await task(async()=>{
  if(!session.value||!q.value)return;
  const view=await api<View>(`/sessions/${session.value.id}/answers`,{questionId:q.value.id,answer:skip?'':answer.value.trim(),skip});
  await applyView(view);
});}
async function practice(){await task(async()=>{
  if(!session.value||!selectedSkill.value)return;
  await applyView(await api<View>(`/sessions/${session.value.id}/follow-up`,{skillId:selectedSkill.value}));
});}
function home(){page.value='home';error.value='';pendingFeedback.value=null;}
function setOnline(){online.value=navigator.onLine;}
async function initialize(){loading.value=true;error.value='';try{
  const catalog=await api<{modules:Module[];skills:Skill[]}>('/modules');
  modules.value=catalog.modules;skills.value=catalog.skills;selected.value=catalog.modules.map(m=>m.id);
  latest.value=await api<View|null>('/sessions/latest');
  if(latest.value&&!latest.value.done)await applyView(latest.value);
}catch(e){error.value=e instanceof Error?e.message:'加载失败';}finally{loading.value=false;}}
onMounted(()=>{window.addEventListener('online',setOnline);window.addEventListener('offline',setOnline);initialize();});
onUnmounted(()=>{window.removeEventListener('online',setOnline);window.removeEventListener('offline',setOnline);});
</script>

<template>
  <div class="app-shell">
    <aside class="sidebar">
      <button class="brand" @click="home" aria-label="小步数学首页"><span class="brand-mark"><i></i><i></i><i></i><i></i></span><span>小步<span class="brand-light">数学</span></span></button>
      <div class="grade-label">给五年级的你</div>
      <nav aria-label="主导航">
        <button :class="{active:page!=='result'}" @click="home"><span class="nav-icon">▦</span> 数学小体检 <span class="nav-dot" v-if="page!=='result'"></span></button>
        <button :class="{active:page==='result'}" @click="latest?resume():home()"><span class="nav-icon">⌁</span> 我的学习路线 <span class="nav-dot" v-if="page==='result'"></span></button>
      </nav>
      <div class="sidebar-note"><svg viewBox="0 0 100 90" aria-hidden="true"><path d="M23 76C18 49 42 55 46 38S70 27 76 10" fill="none" stroke="#a5bdaa" stroke-width="2" stroke-dasharray="4 5"/><circle cx="23" cy="76" r="6" fill="#d9e7bd"/><circle cx="46" cy="38" r="6" fill="#d9e7bd"/><path d="m67 14 9-4 1 10" fill="none" stroke="#738d70" stroke-width="2"/></svg><p>每一小步，<br>都算数。</p><span>按自己的节奏就好</span></div>
      <button class="source-link" @click="sourceDialog?.showModal()">关于小步与内容来源 ↗</button>
    </aside>

    <main>
      <header class="topbar"><span>{{ page==='home'?'发现 · 练习 · 再试一次':page==='question'?phaseLabel:'我的学习路线' }}</span><span class="top-pill"><i></i> 小学 · 五年级</span></header>
      <div v-if="!online" class="notice" role="status">连接暂时中断。已提交的进度会保留，恢复网络后再继续。</div>
      <div v-if="error" class="error" role="alert">{{ error }} <button v-if="!modules.length" @click="initialize">重新加载</button><button v-else @click="error=''" aria-label="关闭提示">×</button></div>
      <div v-if="loading" class="loading" role="status"><span class="loading-dot"></span> 正在准备你的小体检…</div>

      <template v-else-if="page==='home'">
        <section class="welcome">
          <div class="eyebrow"><span></span> 数学，不着急</div>
          <h1>找到你的<br><em>下一小步。</em></h1>
          <p class="intro">会的，给自己一点肯定。<br>还不太会的，我们一起找到从哪里开始。</p>
          <button class="primary hero-cta" @click="start" :disabled="busy||!selected.length">{{ busy?'正在准备…':'开始小体检' }} <span>↗</span></button>
          <div class="hero-fine">最多 10 题 · 不限时 · 不用注册</div>
          <div class="hero-art" aria-hidden="true">
            <span class="art-sticker sticker-one">想一想 ✦</span>
            <div class="math-paper"><div class="paper-top"><span>MY LITTLE STEPS</span><i>✧</i></div><div class="paper-equation">0.8 <span>?</span> 0.75</div><div class="paper-numberline"><i v-for="n in 9" :key="n"></i><b>0</b><b>1</b><span class="number-dot"></span></div><div class="paper-note">原来，我可以这样想。</div><div class="paper-check">✓ <span>又清楚了一点点</span></div></div>
            <span class="art-sticker sticker-two">每一步都有方向 ↗</span><span class="art-star">✳</span>
          </div>
        </section>

        <button v-if="latest" class="resume-card" @click="resume" :disabled="busy"><span class="resume-icon">{{latest.done?'✓':'↻'}}</span><span><strong>{{latest.done?'上次的小发现，还在这里':'接着上次的小体检'}}</strong><small>{{latest.done?'回看学习线索，或选一个知识点练习':'进度已保存，按照你的节奏继续'}}</small></span><span class="resume-arrow">→</span></button>

        <section class="module-section">
          <div class="section-heading"><div><h2>这次，想看看哪些内容？</h2><p>选择学过的模块；遇到没学过的题，可以直接跳过。</p></div><button class="text-button" @click="selected=selected.length===modules.length?[]:modules.map(m=>m.id)">{{selected.length===modules.length?'清空选择':'选择全部'}}</button></div>
          <div class="module-grid">
            <button v-for="(m,i) in modules" :key="m.id" class="module-card" :class="{selected:selected.includes(m.id)}" :aria-pressed="selected.includes(m.id)" @click="toggle(m.id)">
              <span class="module-symbol" :class="`symbol-${i}`">{{m.symbol}}</span><span class="check-box">{{selected.includes(m.id)?'✓':''}}</span><span class="module-number">{{m.number}} /</span><h3>{{m.name}}</h3><p>{{m.description}}</p><span class="module-count">2 个起步知识点</span>
            </button>
          </div>
          <div class="selection-footer"><span>已选 <strong>{{selected.length}}</strong> 个模块 <span class="muted">· 每个模块先从一道题开始</span></span><button class="primary" @click="start" :disabled="busy||!selected.length">出发，试试看 <span>→</span></button></div>
        </section>
        <footer class="home-footer"><span>✧ 这里只和自己的上一次比较</span><span>一次小体检，不定义你的数学能力</span><button class="text-button" @click="sourceDialog?.showModal()">关于与来源 ↗</button></footer>
      </template>

      <section v-else-if="page==='question'&&session" class="question-layout">
        <div class="question-meta"><button class="back-link" @click="home">← 稍后继续</button><span>{{session.phase==='screen'||session.phase==='confirm'?`已完成 ${session.step} 题 · 最多 ${session.total} 题`:`${session.step} / ${session.total} 已完成`}}</span></div>
        <div class="progress-track" role="progressbar" :aria-valuenow="session.step" :aria-valuemax="session.total" aria-label="本轮进度"><span :style="{width:progress+'%'}"></span></div>
        <template v-if="pendingFeedback">
          <div class="question-card feedback-card" aria-live="polite"><span class="feedback-emblem">{{pendingFeedback.correct?'✓':'✧'}}</span><div class="eyebrow">练习中的小发现</div><h1>{{pendingFeedback.correct?'这个思路，很棒。':'换个角度，就清楚一点。'}}</h1><p class="explanation">{{pendingFeedback.explanation}}</p><p class="quiet-note">练习是用来理解的。接下来用新题再试一试。</p><button class="primary" @click="pendingFeedback=null">{{session.phase==='retest'?'准备好了，换题试试':'继续下一题'}} <span>→</span></button></div>
        </template>
        <form v-else-if="q" class="question-card" @submit.prevent="submit()" :key="q.id">
          <div class="question-tags"><span class="subject-tag">{{moduleName(session.skill?.moduleId||'')}}</span><span>{{phaseLabel}}</span></div>
          <h1 ref="questionHeading" tabindex="-1" class="question-title">{{q.prompt}}</h1>
          <div v-if="q.chart" class="chart" role="img" :aria-label="q.chart.map(b=>`${b.label}${b.value}千克`).join('，')"><div class="chart-title">收集废纸的质量 <span>单位：千克</span></div><div v-for="bar in q.chart" class="bar-row" :key="bar.label"><span>{{bar.label}}</span><div class="bar-track"><i :style="{width:bar.value/Math.max(...q.chart.map(b=>b.value))*88+'%'}"></i><b>{{bar.value}}</b></div></div><p>每根条形右侧标有对应的准确数值</p></div>
          <svg v-if="q.figure" class="geometry-figure" viewBox="0 0 350 190" role="img" :aria-label="q.prompt">
            <template v-if="q.figure.type==='triangle'"><path d="M65 135L280 135L125 35Z" fill="#e4edce" stroke="#6c875b" stroke-width="2"/><path d="M125 35V135M125 124H136V135" fill="none" stroke="#6c875b" stroke-width="1.5" stroke-dasharray="4 3"/><text x="160" y="160">底 {{q.figure.base}} cm</text><text x="25" y="85">高 {{q.figure.height}} cm</text></template>
            <template v-else-if="q.figure.type==='parallelogram'"><path d="M65 135H255L300 45H110Z" fill="#e4edce" stroke="#6c875b" stroke-width="2"/><path d="M110 45V135M110 124H121V135" fill="none" stroke="#6c875b" stroke-dasharray="4 3"/><text x="155" y="160">底 {{q.figure.base}} cm</text><text x="20" y="85">高 {{q.figure.height}} cm</text></template>
            <template v-else><path d="M85 75L215 75L270 40L140 40Z" fill="#edf1dc" stroke="#6c875b" stroke-width="2"/><path d="M215 75L270 40V120L215 155Z" fill="#c9d9ae" stroke="#6c875b" stroke-width="2"/><path d="M85 75H215V155H85Z" fill="#e0eacb" stroke="#6c875b" stroke-width="2"/><text x="130" y="178">长 {{q.figure.base}} cm</text><text x="252" y="151">宽 {{q.figure.width}} cm</text><text x="5" y="119">高 {{q.figure.height}} cm</text></template>
            <text x="265" y="185" class="figure-note">示意图，非按比例</text>
          </svg>
          <div v-if="q.kind==='choice'" class="options" role="group" aria-label="答案选项"><button type="button" v-for="(option,i) in q.options" :key="option.id" class="option" :class="{chosen:answer===option.id}" :aria-pressed="answer===option.id" @click="answer=option.id"><span class="option-letter">{{String.fromCharCode(65+i)}}</span><span>{{option.text}}</span><span v-if="answer===option.id" class="option-check">✓</span></button></div>
          <div v-else class="number-answer"><label for="answer-input">{{q.kind==='fraction'?'写下完整分数':'写下你的答案'}}</label><div class="input-wrap"><input id="answer-input" v-model="answer" :inputmode="q.kind==='fraction'?'text':'decimal'" autocomplete="off" maxlength="80" :placeholder="q.kind==='fraction'?'例如 2/3':'在这里输入数字'" :disabled="busy"><span v-if="q.unit">{{q.unit}}</span></div><small>{{q.kind==='fraction'?'用 / 分隔分子和分母，按题目要求作答':'数字即可，不需要填写单位'}}</small></div>
          <div v-if="session.phase==='practice'" class="practice-tip"><span>✧ 小提示</span><p>{{session.skill?.tip}}</p></div>
          <div class="question-actions"><button type="button" class="text-button" @click="submit('dont_know')" :disabled="busy">暂时不会</button><button type="submit" class="primary" :disabled="!canSubmit">{{busy?'正在保存…':session.phase==='practice'?'看看思路':'提交答案'}} <span>→</span></button></div>
          <button type="button" class="not-learned" @click="submit('not_learned')" :disabled="busy">这个我还没学过，先跳过</button>
        </form>
        <div v-else class="question-card feedback-card"><h1>作答已经保存。</h1><p class="quiet-note">连接恢复后，就可以查看这次的小发现。</p><button class="primary" @click="task(showResult)" :disabled="busy">查看本次结果 →</button></div>
        <p class="question-reassurance">{{session.phase==='practice'?'可以借助提示慢慢想，理解比速度重要。':session.phase==='retest'?'这是没见过的新题，试着自己完成。':'认真想过就很好，不确定也没有关系。'}}</p>
      </section>

      <template v-else-if="page==='result'&&result">
        <section class="result-heading"><div class="eyebrow"><span></span> 你的小发现</div><h1>下一步，<em>更清楚了。</em></h1><p>这是一份本次观察，不是能力标签。挑一个知识点，再向前走一小步。</p></section>
        <div class="result-stats"><div><strong>{{result.tested}}</strong><span>个知识点已有独立作答</span></div><div><strong>{{result.unseen}}</strong><span>个知识点还没有作答证据</span></div><div><strong>{{passed}}</strong><span>个知识点本次复测通过</span></div></div>
        <div class="result-columns">
          <section class="next-step" v-if="activeEvidence"><div class="eyebrow">{{activeEvidence.practiced?'这一小步，已经走完':'这一小步，可以从这里开始'}}</div><h2>{{activeEvidence.skill.name}}</h2><span class="status-badge" :class="activeEvidence.status">{{activeEvidence.practiced?'初测：':''}}{{labels[activeEvidence.status]}}</span><p>{{evidenceText(activeEvidence)}}</p><p class="skill-tip">{{activeEvidence.skill.tip}}</p><div v-if="activeEvidence.retestCount" class="retest-note">{{activeEvidence.retestCorrect===2?'✓ 本次复测通过':'✧ 复测后还值得再巩固'}}<small>新题 {{activeEvidence.retestCount}} 道 · 正确 {{activeEvidence.retestCorrect}} 道。以后再用新题看看是否记得。</small></div><button v-if="!activeEvidence.practiced&&activeEvidence.status!=='outside'" class="primary" @click="practice" :disabled="busy">{{busy?'准备中…':'练两题，再试一次'}} <span>→</span></button><p v-else class="quiet-note">{{activeEvidence.practiced?'这轮练习已完成，可以选择其他知识点。':'这个模块不在本次范围内，下次可以选择它。'}}</p></section>
          <section class="route-card"><div class="section-heading"><h2>知识是怎样连起来的？</h2><span class="small-tag">小路线</span></div><div v-if="route" class="route-list"><div v-for="node in routeNodes" :key="node.id" class="route-node" :class="{current:node.current}"><i>{{node.current?'✦':'○'}}</i><div><small>{{node.tag}}</small><strong>{{node.name}}</strong><span>{{node.current?'本次关注的能力点':'关系参考 · 未单独检测'}}</span></div></div></div><p v-else class="quiet-note">正在展开路线…</p><p class="route-note">基础知识值得回看；仅凭关联，还不能判断答错的原因。</p></section>
        </div>
        <section class="observations"><div class="section-heading"><div><h2>这次看到的，和还没看到的</h2><p>点一个知识点，看看它的小路线。</p></div><span class="small-tag">{{result.skills.length}} 个起步知识点</span></div><div class="evidence-grid"><button v-for="e in result.skills" :key="e.skill.id" class="evidence-card" :class="{focused:selectedSkill===e.skill.id}" @click="task(()=>chooseRoute(e.skill.id))"><small>{{moduleName(e.skill.moduleId)}}</small><div><h3>{{e.skill.name}}</h3><span>↗</span></div><span class="status-badge" :class="e.status">{{e.practiced?'初测：':''}}{{labels[e.status]}}</span><p>{{evidenceText(e)}}</p><span class="passed-label" v-if="e.retestCount===2">{{e.retestCorrect===2?'✓ 本次复测通过':'复测后仍可继续巩固'}}</span></button></div></section>
        <details class="answer-review"><summary>回看这次的题目与思路 <span>{{result.review.length}} 道</span></summary><article v-for="(r,i) in result.review" :key="r.question.id"><span class="review-index">{{i+1}}</span><div><small>{{r.question.purpose==='practice'?'练习':r.question.purpose==='retest'?'复测':'独立检测'}} · {{r.skip==='not_learned'?'还没学过':r.skip?'暂时不会':r.correct?'回答正确':'值得再想想'}}</small><h3>{{r.question.prompt}}</h3><p v-if="!r.skip">你的答案：{{r.answer}}</p><p class="review-explanation">{{r.explanation}}</p></div></article></details>
        <div class="result-bottom"><p>今天的一小步，就很值得肯定。</p><button class="secondary" @click="home">回到首页 →</button></div>
      </template>
    </main>

    <dialog ref="sourceDialog" class="source-dialog" aria-labelledby="source-title"><button class="dialog-close" @click="sourceDialog?.close()" aria-label="关闭">×</button><div class="eyebrow">关于小步数学</div><h2 id="source-title">给好奇心一点方向。</h2><p>小步数学是一份面向五年级的起步练习，包含 6 个模块、12 个能力点。它不会用一次短测定义孩子的能力，也不代表覆盖了全部课程。</p><p>知识关系来自 Marble Skill Taxonomy（v1）。中文能力点和练习为本项目整理，尚未完成中国具体教材版本和学期的正式对齐；题目与诊断规则仍需教师审阅和实际使用验证。</p><p>不需要真实姓名。进度保存在本站服务的 SQLite 数据库，用当前浏览器的匿名标识关联；换浏览器后不会自动同步。</p><p class="source-license">Marble Skill Taxonomy · © Generative Spark, Inc. (Marble)。数据库：ODbL 1.0；原创内容：CC BY-SA 4.0。第三方课程标准适用各自条款。</p><a href="https://github.com/withmarbleapp/os-taxonomy" target="_blank" rel="noreferrer">查看原项目与来源说明 ↗</a></dialog>
  </div>
</template>
