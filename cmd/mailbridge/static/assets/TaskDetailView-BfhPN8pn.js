const __vite__mapDeps=(i,m=__vite__mapDeps,d=(m.f||(m.f=["assets/tasks-6awqbJWV.js","assets/auth-Bnitlmi4.js"])))=>i.map(i=>d[i]);
import{A as e,E as t,H as n,K as r,L as i,M as a,O as o,R as s,_ as c,d as l,f as u,lt as d,m as f,n as p,ot as m,p as h,s as g,u as ee,v as _,w as v}from"./auth-Bnitlmi4.js";import{c as y,t as b}from"./button-wWfh7m_N.js";import{t as x}from"./select-B292IHcs.js";import{a as S,i as C,l as w,n as te,r as ne,t as T}from"./index-D-CxDQSr.js";import{n as re}from"./tasks-6awqbJWV.js";import{n as E,t as D}from"./inputtext-BuFCIxa-.js";import{t as O}from"./card-D6v1s2W4.js";var k={class:`comment-list`},A={key:0,class:`empty`},j={class:`comment-header`},M={class:`author`},N={class:`date`},P={class:`comment-body`},F=C({__name:`CommentList`,props:{comments:{type:Array,default:()=>[]}},setup(t){function n(e){if(!e)return``;let t=new Date(e);return t.toLocaleDateString(`ru-RU`)+` `+t.toLocaleTimeString(`ru-RU`,{hour:`2-digit`,minute:`2-digit`})}return(r,i)=>(o(),f(`div`,k,[t.comments.length===0?(o(),f(`div`,A,`Нет комментариев`)):h(``,!0),(o(!0),f(g,null,e(t.comments,e=>(o(),f(`div`,{key:e.id,class:m([`comment`,e.direction])},[l(`div`,j,[l(`span`,M,d(e.author),1),l(`span`,N,d(n(e.created_at)),1)]),l(`div`,P,d(e.body),1)],2))),128))]))}},[[`__scopeId`,`data-v-9a231af5`]]),I=w.extend({name:`textarea`,style:`
    .p-textarea {
        font-weight: dt('textarea.font.weight');
        font-size: dt('textarea.font.size');
        color: dt('textarea.color');
        background: dt('textarea.background');
        padding-block: dt('textarea.padding.y');
        padding-inline: dt('textarea.padding.x');
        border: 1px solid dt('textarea.border.color');
        transition:
            background dt('textarea.transition.duration'),
            color dt('textarea.transition.duration'),
            border-color dt('textarea.transition.duration'),
            outline-color dt('textarea.transition.duration'),
            box-shadow dt('textarea.transition.duration');
        appearance: none;
        border-radius: dt('textarea.border.radius');
        outline-color: transparent;
        box-shadow: dt('textarea.shadow');
    }

    .p-textarea:enabled:hover {
        border-color: dt('textarea.hover.border.color');
    }

    .p-textarea:enabled:focus {
        border-color: dt('textarea.focus.border.color');
        box-shadow: dt('textarea.focus.ring.shadow');
        outline: dt('textarea.focus.ring.width') dt('textarea.focus.ring.style') dt('textarea.focus.ring.color');
        outline-offset: dt('textarea.focus.ring.offset');
    }

    .p-textarea.p-invalid {
        border-color: dt('textarea.invalid.border.color');
    }

    .p-textarea.p-variant-filled {
        background: dt('textarea.filled.background');
    }

    .p-textarea.p-variant-filled:enabled:hover {
        background: dt('textarea.filled.hover.background');
    }

    .p-textarea.p-variant-filled:enabled:focus {
        background: dt('textarea.filled.focus.background');
    }

    .p-textarea:disabled {
        opacity: 1;
        background: dt('textarea.disabled.background');
        color: dt('textarea.disabled.color');
    }

    .p-textarea::placeholder {
        color: dt('textarea.placeholder.color');
    }

    .p-textarea.p-invalid::placeholder {
        color: dt('textarea.invalid.placeholder.color');
    }

    .p-textarea-fluid {
        width: 100%;
    }

    .p-textarea-resizable {
        overflow: hidden;
        resize: none;
    }

    .p-textarea-sm {
        font-size: dt('textarea.sm.font.size');
        padding-block: dt('textarea.sm.padding.y');
        padding-inline: dt('textarea.sm.padding.x');
    }

    .p-textarea-lg {
        font-size: dt('textarea.lg.font.size');
        padding-block: dt('textarea.lg.padding.y');
        padding-inline: dt('textarea.lg.padding.x');
    }
`,classes:{root:function(e){var t=e.instance,n=e.props;return[`p-textarea p-component`,{"p-filled":t.$filled,"p-textarea-resizable ":n.autoResize,"p-textarea-sm p-inputfield-sm":n.size===`small`,"p-textarea-lg p-inputfield-lg":n.size===`large`,"p-invalid":t.$invalid,"p-variant-filled":t.$variant===`filled`,"p-textarea-fluid":t.$fluid}]}}}),L={name:`BaseTextarea`,extends:E,props:{autoResize:Boolean},style:I,provide:function(){return{$pcTextarea:this,$parentInstance:this}}};function R(e){"@babel/helpers - typeof";return R=typeof Symbol==`function`&&typeof Symbol.iterator==`symbol`?function(e){return typeof e}:function(e){return e&&typeof Symbol==`function`&&e.constructor===Symbol&&e!==Symbol.prototype?`symbol`:typeof e},R(e)}function z(e,t,n){return(t=B(t))in e?Object.defineProperty(e,t,{value:n,enumerable:!0,configurable:!0,writable:!0}):e[t]=n,e}function B(e){var t=V(e,`string`);return R(t)==`symbol`?t:t+``}function V(e,t){if(R(e)!=`object`||!e)return e;var n=e[Symbol.toPrimitive];if(n!==void 0){var r=n.call(e,t);if(R(r)!=`object`)return r;throw TypeError(`@@toPrimitive must return a primitive value.`)}return(t===`string`?String:Number)(e)}var H={name:`Textarea`,extends:L,inheritAttrs:!1,observer:null,mounted:function(){var e=this;this.autoResize&&(this.observer=new ResizeObserver(function(){requestAnimationFrame(function(){e.resize()})}),this.observer.observe(this.$el))},updated:function(){this.autoResize&&this.resize()},beforeUnmount:function(){this.observer&&this.observer.disconnect()},methods:{resize:function(){if(this.$el.offsetParent){var e=this.$el.style.height,t=parseInt(e)||0,n=this.$el.scrollHeight;t&&n<t?(this.$el.style.height=`auto`,this.$el.style.height=`${this.$el.scrollHeight}px`):(!t||n>t)&&(this.$el.style.height=`${n}px`)}},onInput:function(e){this.autoResize&&this.resize(),this.writeValue(e.target.value,e)}},computed:{attrs:function(){return v(this.ptmi(`root`,{context:{filled:this.$filled,disabled:this.disabled}}),this.formField)},dataP:function(){return y(z({invalid:this.$invalid,fluid:this.$fluid,filled:this.$variant===`filled`},this.size,this.size))}}},U=[`value`,`name`,`disabled`,`aria-invalid`,`data-p`];function W(e,t,n,r,i,a){return o(),f(`textarea`,v({class:e.cx(`root`),value:e.d_value,name:e.name,disabled:e.disabled,"aria-invalid":e.invalid||void 0,"data-p":a.dataP,onInput:t[0]||=function(){return a.onInput&&a.onInput.apply(a,arguments)}},a.attrs),null,16,U)}H.render=W;var G={class:`reply-form`},K=C({__name:`ReplyForm`,props:{taskId:{type:Number,required:!0}},emits:[`sent`],setup(e,{emit:t}){let i=e,a=t,s=n(``),c=n(!1);async function l(){if(s.value.trim()){c.value=!0;try{let{useTasksStore:e}=await T(async()=>{let{useTasksStore:e}=await import(`./tasks-6awqbJWV.js`).then(e=>e.t);return{useTasksStore:e}},__vite__mapDeps([0,1]));await e().replyTask(i.taskId,s.value),s.value=``,a(`sent`)}finally{c.value=!1}}}return(e,t)=>(o(),f(`div`,G,[_(r(H),{modelValue:s.value,"onUpdate:modelValue":t[0]||=e=>s.value=e,rows:`3`,placeholder:`Текст ответа...`},null,8,[`modelValue`]),_(r(b),{label:`Отправить`,icon:`pi pi-send`,onClick:l,loading:c.value},null,8,[`loading`])]))}},[[`__scopeId`,`data-v-f04bd7f5`]]),q={class:`workflow-buttons`},J=C({__name:`WorkflowButtons`,props:{currentStatus:{type:String,required:!0}},emits:[`transition`],setup(t){let n=t,i={new:[{status:`in_progress`,label:`В работу`,severity:`primary`,icon:`pi pi-play`},{status:`backlog`,label:`В бэклог`,severity:`secondary`,icon:`pi pi-inbox`},{status:`closed`,label:`Закрыть`,severity:`danger`,icon:`pi pi-times`}],in_progress:[{status:`backlog`,label:`В бэклог`,severity:`secondary`,icon:`pi pi-inbox`},{status:`completed`,label:`Выполнено`,severity:`success`,icon:`pi pi-check`}],backlog:[{status:`in_progress`,label:`В работу`,severity:`primary`,icon:`pi pi-play`},{status:`closed`,label:`Закрыть`,severity:`danger`,icon:`pi pi-times`}],completed:[{status:`closed`,label:`Закрыть`,severity:`danger`,icon:`pi pi-times`},{status:`in_progress`,label:`Вернуть в работу`,severity:`warn`,icon:`pi pi-replay`}],closed:[{status:`in_progress`,label:`Вернуть в работу`,severity:`warn`,icon:`pi pi-replay`}]},a=ee(()=>i[n.currentStatus]||[]);return(t,n)=>(o(),f(`div`,q,[(o(!0),f(g,null,e(a.value,e=>(o(),u(r(b),{key:e.status,label:e.label,severity:e.severity,icon:e.icon,size:`small`,onClick:n=>t.$emit(`transition`,e.status)},null,8,[`label`,`severity`,`icon`,`onClick`]))),128))]))}},[[`__scopeId`,`data-v-42f90af0`]]),Y={key:0,class:`task-detail`},X={class:`task-header`},ie={class:`task-grid`},ae={class:`task-main`},Z={class:`task-meta`},oe=[`innerHTML`],se={key:0,class:`attachments`},ce=[`href`],le={class:`size`},ue={class:`inbox-context-header`},de={key:0},fe={class:`inbox-context-meta`},pe={class:`inbox-context-subject`},me=[`innerHTML`],he={key:1},ge={class:`inbox-context-meta`},_e={class:`inbox-context-subject`},ve=[`innerHTML`],ye={class:`reply-section`},be={class:`task-sidebar`},xe={class:`field`},Se={class:`field`},Ce={class:`field`},we={class:`field`},Te={class:`field`},Q=C({__name:`TaskDetailView`,setup(ee){let v=te(),y=ne(),C=S(),w=re(),T=n(null),E=n(null),k=n(null),A=n(null),j=n(``),M=n([]),N=n([]),P=n(!1),I=[{label:`Входящие`,value:`Входящие`},{label:`ТРК`,value:`ТРК`},{label:`Отель`,value:`Отель`},{label:`Лидер Спорт`,value:`Лидер Спорт`},{label:`Театр`,value:`Театр`},{label:`Мебельный центр`,value:`Мебельный центр`},{label:`Кафе`,value:`Кафе`},{label:`Ледовая арена`,value:`Ледовая арена`},{label:`Корпоративные сайты`,value:`Корпоративные сайты`}],L=[{label:`Новая`,value:`new`},{label:`Бэклог`,value:`backlog`},{label:`В работе`,value:`in_progress`},{label:`Выполнена`,value:`completed`},{label:`Закрыта`,value:`closed`}],R=[{label:`Urgent`,value:`urgent`},{label:`High`,value:`high`},{label:`Medium`,value:`medium`},{label:`Low`,value:`low`}],z=[{label:`Bug`,value:`bug`},{label:`Feature`,value:`feature`},{label:`Support`,value:`support`},{label:`Access`,value:`access`},{label:`SEO`,value:`seo`},{label:`Content`,value:`content`}];t(async()=>{await w.fetchTask(v.params.id),B(),w.markAsRead(v.params.id),M.value=await w.fetchTaskInbox(v.params.id),N.value=await U(v.params.id)}),i(()=>w.currentTask,B);function B(){w.currentTask&&(T.value=w.currentTask.project,E.value=w.currentTask.status,k.value=w.currentTask.priority,A.value=w.currentTask.type,j.value=w.currentTask.assignee)}async function V(e,t){await w.updateTask(v.params.id,{[e]:t})}async function H(e){E.value=e,await V(`status`,e)}async function U(e){try{let{data:t}=await p.get(`/tasks/${e}/attachments`);return t}catch{return[]}}async function W(e){try{await p.delete(`/tasks/${v.params.id}/attachments/${e}`),N.value=N.value.filter(t=>t.id!==e),C.add({severity:`success`,summary:`Вложение откреплено`,life:2e3})}catch{C.add({severity:`error`,summary:`Ошибка`,life:3e3})}}function G(e){let t=e.body_html||$(e.body_text);return t.length>5e3?t.slice(0,5e3)+`...`:t}function q(e){return(e.body_html||$(e.body_text)).length>2e3}function Q(e){e.target.tagName===`IMG`&&e.target.src&&window.open(e.target.src,`_blank`)}function Ee(){}function De(){let e=v.query.tab;y.push({path:`/`,query:e?{tab:e}:{}})}function Oe(e){if(!e)return``;let t=new Date(e);return t.toLocaleDateString(`ru-RU`)+` `+t.toLocaleTimeString(`ru-RU`,{hour:`2-digit`,minute:`2-digit`})}function ke(e){return e<1024?e+` B`:e<1048576?(e/1024).toFixed(1)+` KB`:(e/1048576).toFixed(1)+` MB`}function $(e){return e?.replace(/\n/g,`<br>`)||``}return(t,n)=>{let i=a(`router-link`);return r(w).currentTask?(o(),f(`div`,Y,[l(`header`,X,[_(r(b),{icon:`pi pi-arrow-left`,severity:`secondary`,text:``,onClick:De}),l(`h2`,null,`#`+d(r(w).currentTask.id)+` `+d(r(w).currentTask.subject),1)]),l(`div`,ie,[l(`div`,ae,[_(r(O),null,{title:s(()=>[...n[12]||=[c(`Описание`,-1)]]),content:s(()=>[l(`div`,Z,[l(`span`,null,[n[13]||=l(`strong`,null,`От:`,-1),c(` `+d(r(w).currentTask.from_email),1)]),l(`span`,null,[n[14]||=l(`strong`,null,`Дата:`,-1),c(` `+d(Oe(r(w).currentTask.created_at)),1)])]),l(`div`,{class:`task-body`,onClick:Q,innerHTML:r(w).currentTask.body_html||$(r(w).currentTask.body_text)},null,8,oe),N.value.length>0?(o(),f(`div`,se,[n[16]||=l(`h4`,null,`Вложения`,-1),(o(!0),f(g,null,e(N.value,e=>(o(),f(`div`,{key:e.id,class:`attachment-item`},[n[15]||=l(`i`,{class:`pi pi-paperclip`},null,-1),l(`a`,{href:`/api/attachments/${e.storage_path}/${encodeURIComponent(e.filename)}`,target:`_blank`},d(e.filename),9,ce),l(`span`,le,d(ke(e.size)),1),_(r(b),{icon:`pi pi-times`,text:``,size:`small`,severity:`danger`,onClick:t=>W(e.id),title:`Открепить`},null,8,[`onClick`])]))),128))])):h(``,!0)]),footer:s(()=>[_(J,{currentStatus:E.value,onTransition:H},null,8,[`currentStatus`])]),_:1}),M.value.length>0?(o(),u(r(O),{key:0,class:`inbox-context`},{title:s(()=>[l(`div`,ue,[n[17]||=l(`span`,null,`📧 Оригинальное письмо`,-1),l(`i`,{class:m(P.value?`pi pi-chevron-up`:`pi pi-chevron-down`),onClick:n[0]||=e=>P.value=!P.value},null,2)])]),content:s(()=>[P.value?(o(),f(`div`,de,[(o(!0),f(g,null,e(M.value,e=>(o(),f(`div`,{key:e.id,class:`inbox-context-item`},[l(`div`,fe,[l(`span`,pe,d(e.subject),1),_(i,{to:`/inbox/${e.id}`,class:`inbox-link`},{default:s(()=>[...n[18]||=[c(`Открыть в ленте`,-1)]]),_:1},8,[`to`])]),l(`div`,{class:`item-body`,onClick:Q,innerHTML:e.body_html||$(e.body_text)},null,8,me)]))),128))])):(o(),f(`div`,he,[(o(!0),f(g,null,e(M.value,e=>(o(),f(`div`,{key:e.id,class:`inbox-context-item`},[l(`div`,ge,[l(`span`,_e,d(e.subject),1),_(i,{to:`/inbox/${e.id}`,class:`inbox-link`},{default:s(()=>[...n[19]||=[c(`Открыть в ленте`,-1)]]),_:1},8,[`to`])]),l(`div`,{class:`item-body-preview`,innerHTML:G(e)},null,8,ve),q(e)?(o(),u(r(b),{key:0,label:`Показать полностью`,text:``,size:`small`,onClick:n[1]||=e=>P.value=!0,class:`expand-btn`})):h(``,!0)]))),128))]))]),_:1})):h(``,!0),_(r(O),null,{title:s(()=>[...n[20]||=[c(`Комментарии`,-1)]]),content:s(()=>[_(F,{comments:r(w).currentComments},null,8,[`comments`]),l(`div`,ye,[_(K,{taskId:r(w).currentTask.id,onSent:Ee},null,8,[`taskId`])])]),_:1})]),l(`div`,be,[_(r(O),null,{content:s(()=>[l(`div`,xe,[n[21]||=l(`label`,null,`Проект`,-1),_(r(x),{modelValue:T.value,"onUpdate:modelValue":n[2]||=e=>T.value=e,options:I,optionLabel:`label`,optionValue:`value`,onChange:n[3]||=e=>V(`project`,e.value)},null,8,[`modelValue`])]),l(`div`,Se,[n[22]||=l(`label`,null,`Статус`,-1),_(r(x),{modelValue:E.value,"onUpdate:modelValue":n[4]||=e=>E.value=e,options:L,optionLabel:`label`,optionValue:`value`,onChange:n[5]||=e=>V(`status`,e.value)},null,8,[`modelValue`])]),l(`div`,Ce,[n[23]||=l(`label`,null,`Приоритет`,-1),_(r(x),{modelValue:k.value,"onUpdate:modelValue":n[6]||=e=>k.value=e,options:R,optionLabel:`label`,optionValue:`value`,onChange:n[7]||=e=>V(`priority`,e.value)},null,8,[`modelValue`])]),l(`div`,we,[n[24]||=l(`label`,null,`Тип`,-1),_(r(x),{modelValue:A.value,"onUpdate:modelValue":n[8]||=e=>A.value=e,options:z,optionLabel:`label`,optionValue:`value`,onChange:n[9]||=e=>V(`type`,e.value)},null,8,[`modelValue`])]),l(`div`,Te,[n[25]||=l(`label`,null,`Исполнитель`,-1),_(r(D),{modelValue:j.value,"onUpdate:modelValue":n[10]||=e=>j.value=e,onBlur:n[11]||=e=>V(`assignee`,j.value)},null,8,[`modelValue`])])]),_:1})])])])):h(``,!0)}}},[[`__scopeId`,`data-v-bb94c1d3`]]);export{Q as default};