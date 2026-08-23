import{A as e,D as t,E as n,G as r,H as i,K as a,L as o,M as s,N as c,O as l,P as u,R as d,_ as f,d as p,f as m,i as h,lt as g,m as _,n as ee,ot as v,p as y,s as b,st as x,t as te,u as ne,v as S,w as C,x as w,y as T,z as E}from"./auth-Bnitlmi4.js";import{c as D,i as re,n as O,o as k,s as A,t as j}from"./button-wWfh7m_N.js";import{c as M,l as ie,t as ae,u as N}from"./select-B292IHcs.js";import{I as oe,a as se,i as P,l as ce,n as F,o as I,ot as L,pt as le,r as R}from"./index-D-CxDQSr.js";import{n as z}from"./tasks-6awqbJWV.js";import{r as ue,t as de}from"./inputtext-BuFCIxa-.js";import{t as B}from"./tag-CDFFpYCL.js";import fe,{n as pe,r as me,t as V}from"./InboxView-DoGno0TQ.js";var he=h(`theme`,()=>{let e=i(localStorage.getItem(`mailbridge_theme`)===`dark`);function t(){e.value?document.documentElement.classList.add(`dark`):document.documentElement.classList.remove(`dark`)}function n(){e.value=!e.value,localStorage.setItem(`mailbridge_theme`,e.value?`dark`:`light`),t()}return t(),{isDark:e,toggleTheme:n}}),ge=`
    .p-toast {
        width: dt('toast.width');
        white-space: pre-line;
        word-break: break-word;
    }

    .p-toast-message {
        --px-offset-y: calc(var(--px-swipe-amount-y) + (var(--px-toast-offset) + var(--px-toast-index) * var(--px-gap)) * var(--px-raise-factor));
        --px-offset-x: var(--px-swipe-amount-x);
        width: 100%;
        outline: none;
        position: absolute;
        touch-action: none;
        opacity: 0;
        transform: translateX(var(--px-offset-x)) translateY(calc(100% * var(--px-raise-factor) * -1));
        z-index: var(--px-toast-z-index);
        transition: transform dt('toast.transition.duration'), opacity dt('toast.transition.duration'), height dt('toast.transition.duration');
    }

    .p-toast-message:focus-visible {
        box-shadow: dt('toast.focus.ring.shadow');
        outline: dt('toast.focus.ring.width') dt('toast.focus.ring.style') dt('focus.ring.color');
        outline-offset: dt('toast.focus.ring.offset');
    }

    .p-toast-message[data-mounted] {
        opacity: 1;
        transform: translateY(0);
    }

    .p-toast-message:not([data-expanded]):not([data-front]) {
        overflow: hidden;
        height: var(--px-front-toast-height);
        transform: translateX(var(--px-offset-x)) translateY(calc(var(--px-raise-factor) * var(--px-toast-index) * var(--px-gap))) scale(calc(var(--px-toast-index) * -0.05 + 1));
    }

    .p-toast-message[data-mounted][data-expanded] {
        height: var(--px-initial-height);
        transform: translateX(var(--px-offset-x)) translateY(var(--px-offset-y));
    }

    .p-toast-message[data-expanded]::after {
        content: "";
        position: absolute;
        left: 0;
        height: calc(var(--px-gap) + 1px);
        width: 100%;
        bottom: 100%;
    }

    .p-toast-message:not([data-visible]) {
        opacity: 0;
        pointer-events: none;
        user-select: none;
    }

    .p-toast-message[data-removed][data-front]:not([data-swipe-out]) {
        opacity: 0;
        transform: translateX(var(--px-offset-x)) translateY(calc(var(--px-raise-factor) * -100%));
    }

    .p-toast-message[data-removed]:not([data-front]):not([data-swipe-out])[data-expanded] {
        opacity: 0;
        transform: translateX(var(--px-offset-x)) translateY(calc((var(--px-offset-y)) + (var(--px-raise-factor) * -100%)));
    }

    .p-toast-message[data-removed]:not([data-front]):not([data-swipe-out]):not([data-expanded]) {
        opacity: 0;
        transform: translateX(var(--px-offset-x)) translateY(calc(var(--px-raise-factor) * 40% * -1));
        transition:
            transform 500ms,
            opacity 200ms;
    }

    .p-toast-message[data-swiping] {
        transition: none;
        transform: translateX(var(--px-offset-x)) translateY(var(--px-offset-y)) !important;
    }

    .p-toast-message[data-swiped] {
        -webkit-user-select: none;
        user-select: none;
    }

    .p-toast-message[data-swipe-out][data-swipe-direction="up"] {
        opacity: 0;
        transform: translateX(var(--px-offset-x)) translateY(calc(var(--px-offset-y) - 100%)) !important;
    }

    .p-toast-message[data-swipe-out][data-swipe-direction="down"] {
        opacity: 0;
        transform: translateX(var(--px-offset-x)) translateY(calc(var(--px-offset-y) + 100%)) !important;
    }

    .p-toast-message[data-swipe-out][data-swipe-direction="left"] {
        opacity: 0;
        transform: translateX(calc(var(--px-offset-x) - 100%)) translateY(var(--px-offset-y)) !important;
    }

    .p-toast-message[data-swipe-out][data-swipe-direction="right"] {
        opacity: 0;
        transform: translateX(calc(var(--px-offset-x) + 100%)) translateY(var(--px-offset-y)) !important;
        transition:
            transform 500ms,
            opacity 200ms;
    }

    .p-toast-message-icon,
    .p-toast-message-icon svg,
    .p-toast-message-icon i {
        flex-shrink: 0;
        font-size: dt('toast.icon.size');
        width: dt('toast.icon.size');
        height: dt('toast.icon.size');
        margin: dt('toast.icon.margin');
    }

    .p-toast-message-content {
        display: flex;
        align-items: flex-start;
        padding: dt('toast.content.padding');
        gap: dt('toast.content.gap');
        min-height: 0;
        overflow: hidden;
        transition: padding 250ms ease-in;
    }

    .p-toast-message-text {
        flex: 1 1 auto;
        display: flex;
        flex-direction: column;
        gap: dt('toast.text.gap');
    }

    .p-toast-summary {
        font-weight: dt('toast.summary.font.weight');
        font-size: dt('toast.summary.font.size');
    }

    .p-toast-detail {
        font-weight: dt('toast.detail.font.weight');
        font-size: dt('toast.detail.font.size');
    }

    .p-toast-close-button {
        display: flex;
        align-items: center;
        justify-content: center;
        overflow: hidden;
        position: absolute;
        cursor: pointer;
        background: transparent;
        transition:
            background dt('toast.transition.duration'),
            color dt('toast.transition.duration'),
            outline-color dt('toast.transition.duration'),
            box-shadow dt('toast.transition.duration');
        outline-color: transparent;
        color: inherit;
        width: dt('toast.close.button.width');
        height: dt('toast.close.button.height');
        border-radius: dt('toast.close.button.border.radius');
        margin: 0;
        top: 0.25rem;
        right: 0.25rem;
        padding: 0;
        border: none;
        user-select: none;
    }

    .p-toast-close-button:dir(rtl) {
        left: 0.25rem;
        right: auto;
    }

    .p-toast-message-normal,
    .p-toast-message-info,
    .p-toast-message-success,
    .p-toast-message-warn,
    .p-toast-message-error,
    .p-toast-message-secondary,
    .p-toast-message-contrast {
        border-width: dt('toast.border.width');
        border-style: solid;
        backdrop-filter: blur(dt('toast.blur'));
        border-radius: dt('toast.border.radius');
    }

    .p-toast-close-icon,
    .p-toast-close-icon svg,
    .p-toast-close-icon i {
        font-size: dt('toast.close.icon.size');
        width: dt('toast.close.icon.size');
        height: dt('toast.close.icon.size');
    }

    .p-toast-close-button:focus-visible {
        outline-width: dt('focus.ring.width');
        outline-style: dt('focus.ring.style');
        outline-offset: dt('focus.ring.offset');
    }

    .p-toast-message-normal {
        background: dt('toast.normal.background');
        border-color: dt('toast.normal.border.color');
        color: dt('toast.normal.color');
        box-shadow: dt('toast.normal.shadow');
    }

    .p-toast-message-normal .p-toast-detail {
        color: dt('toast.normal.detail.color');
    }

    .p-toast-message-normal .p-toast-close-button:focus-visible {
        outline-color: dt('toast.normal.close.button.focus.ring.color');
        box-shadow: dt('toast.normal.close.button.focus.ring.shadow');
    }

    .p-toast-message-normal .p-toast-close-button:hover {
        background: dt('toast.normal.close.button.hover.background');
    }

    .p-toast-message-info {
        background: dt('toast.info.background');
        border-color: dt('toast.info.border.color');
        color: dt('toast.info.color');
        box-shadow: dt('toast.info.shadow');
    }

    .p-toast-message-info .p-toast-detail {
        color: dt('toast.info.detail.color');
    }

    .p-toast-message-info .p-toast-close-button:focus-visible {
        outline-color: dt('toast.info.close.button.focus.ring.color');
        box-shadow: dt('toast.info.close.button.focus.ring.shadow');
    }

    .p-toast-message-info .p-toast-close-button:hover {
        background: dt('toast.info.close.button.hover.background');
    }

    .p-toast-message-success {
        background: dt('toast.success.background');
        border-color: dt('toast.success.border.color');
        color: dt('toast.success.color');
        box-shadow: dt('toast.success.shadow');
    }

    .p-toast-message-success .p-toast-detail {
        color: dt('toast.success.detail.color');
    }

    .p-toast-message-success .p-toast-close-button:focus-visible {
        outline-color: dt('toast.success.close.button.focus.ring.color');
        box-shadow: dt('toast.success.close.button.focus.ring.shadow');
    }

    .p-toast-message-success .p-toast-close-button:hover {
        background: dt('toast.success.close.button.hover.background');
    }

    .p-toast-message-warn {
        background: dt('toast.warn.background');
        border-color: dt('toast.warn.border.color');
        color: dt('toast.warn.color');
        box-shadow: dt('toast.warn.shadow');
    }

    .p-toast-message-warn .p-toast-detail {
        color: dt('toast.warn.detail.color');
    }

    .p-toast-message-warn .p-toast-close-button:focus-visible {
        outline-color: dt('toast.warn.close.button.focus.ring.color');
        box-shadow: dt('toast.warn.close.button.focus.ring.shadow');
    }

    .p-toast-message-warn .p-toast-close-button:hover {
        background: dt('toast.warn.close.button.hover.background');
    }

    .p-toast-message-error {
        background: dt('toast.error.background');
        border-color: dt('toast.error.border.color');
        color: dt('toast.error.color');
        box-shadow: dt('toast.error.shadow');
    }

    .p-toast-message-error .p-toast-detail {
        color: dt('toast.error.detail.color');
    }

    .p-toast-message-error .p-toast-close-button:focus-visible {
        outline-color: dt('toast.error.close.button.focus.ring.color');
        box-shadow: dt('toast.error.close.button.focus.ring.shadow');
    }

    .p-toast-message-error .p-toast-close-button:hover {
        background: dt('toast.error.close.button.hover.background');
    }

    .p-toast-message-secondary {
        background: dt('toast.secondary.background');
        border-color: dt('toast.secondary.border.color');
        color: dt('toast.secondary.color');
        box-shadow: dt('toast.secondary.shadow');
    }

    .p-toast-message-secondary .p-toast-detail {
        color: dt('toast.secondary.detail.color');
    }

    .p-toast-message-secondary .p-toast-close-button:focus-visible {
        outline-color: dt('toast.secondary.close.button.focus.ring.color');
        box-shadow: dt('toast.secondary.close.button.focus.ring.shadow');
    }

    .p-toast-message-secondary .p-toast-close-button:hover {
        background: dt('toast.secondary.close.button.hover.background');
    }

    .p-toast-message-contrast {
        background: dt('toast.contrast.background');
        border-color: dt('toast.contrast.border.color');
        color: dt('toast.contrast.color');
        box-shadow: dt('toast.contrast.shadow');
    }
    
    .p-toast-message-contrast .p-toast-detail {
        color: dt('toast.contrast.detail.color');
    }

    .p-toast-message-contrast .p-toast-close-button:focus-visible {
        outline-color: dt('toast.contrast.close.button.focus.ring.color');
        box-shadow: dt('toast.contrast.close.button.focus.ring.shadow');
    }

    .p-toast-message-contrast .p-toast-close-button:hover {
        background: dt('toast.contrast.close.button.hover.background');
    }

    .p-toast {
        position: fixed;
        width: 18.75rem;
        z-index: 2000;
    }

    .p-toast-center {
        left: 50%;
        transform: translateX(-50%) translateY(-50%);
        top: 50%;
    }

    .p-toast-bottom-right {
        right: 2rem;
        bottom: 2rem;
    }

    .p-toast-bottom-center {
        bottom: 2rem;
        left: 50%;
        transform: translateX(-50%);
    }

    .p-toast-bottom-left {
        left: 2rem;
        bottom: 2rem;
    }

    .p-toast-top-right {
        right: 2rem;
        top: 2rem;
    }

    .p-toast-top-center {
        left: 50%;
        transform: translateX(-50%);
        top: 2rem;
    }

    .p-toast-top-left {
        left: 2rem;
        top: 2rem;
    }

    .p-toast-bottom-right .p-toast-message{
        --px-raise-factor: -1;
        bottom: 0;
        right: 0;
    }

    .p-toast-bottom-center .p-toast-message{
        --px-raise-factor: -1;
        bottom: 0;
    }

    .p-toast[data-position="bottom-left"] .p-toast-message{
        --px-raise-factor: -1;
        bottom: 0;
        left: 0;
    }

    .p-toast[data-position="top-right"] .p-toast-message{
        --px-raise-factor: 1;
        top: 0;
        right: 0;
    }

    .p-toast[data-position="top-center"] .p-toast-message{
        --px-raise-factor: 1;
        top: 0;
    }

    .p-toast[data-position="top-left"] .p-toast-message{
        --px-raise-factor: 1;
        top: 0;
        left: 0;
    }

    .p-toast[data-position="center"] .p-toast-message{
        --px-raise-factor: 1;
        top: 0;
    }
`;function H(e){"@babel/helpers - typeof";return H=typeof Symbol==`function`&&typeof Symbol.iterator==`symbol`?function(e){return typeof e}:function(e){return e&&typeof Symbol==`function`&&e.constructor===Symbol&&e!==Symbol.prototype?`symbol`:typeof e},H(e)}function U(e,t,n){return(t=_e(t))in e?Object.defineProperty(e,t,{value:n,enumerable:!0,configurable:!0,writable:!0}):e[t]=n,e}function _e(e){var t=ve(e,`string`);return H(t)==`symbol`?t:t+``}function ve(e,t){if(H(e)!=`object`||!e)return e;var n=e[Symbol.toPrimitive];if(n!==void 0){var r=n.call(e,t);if(H(r)!=`object`)return r;throw TypeError(`@@toPrimitive must return a primitive value.`)}return(t===`string`?String:Number)(e)}var ye=ce.extend({name:`toast`,style:ge,classes:{root:function(e){return[`p-toast p-component`,`p-toast-`+e.props.position]},message:function(e){var t=e.props;return[`p-toast-message`,{"p-toast-message-normal":t.message.severity===`normal`||t.message.severity===void 0,"p-toast-message-info":t.message.severity===`info`,"p-toast-message-warn":t.message.severity===`warn`,"p-toast-message-error":t.message.severity===`error`,"p-toast-message-success":t.message.severity===`success`,"p-toast-message-secondary":t.message.severity===`secondary`,"p-toast-message-contrast":t.message.severity===`contrast`}]},messageContent:`p-toast-message-content`,messageIcon:function(e){var t=e.props;return[`p-toast-message-icon`,U(U(U(U(U(U({},t.infoIcon,t.message.severity===`info`),t.warnIcon,t.message.severity===`warn`),t.errorIcon,t.message.severity===`error`),t.successIcon,t.message.severity===`success`),t.secondaryIcon,t.message.severity===`secondary`),t.contrastIcon,t.message.severity===`contrast`)]},messageText:`p-toast-message-text`,summary:`p-toast-summary`,detail:`p-toast-detail`,closeButton:`p-toast-close-button`,closeIcon:`p-toast-close-icon`},inlineStyles:{root:function(e){var t=e.position;return{position:`fixed`,top:t===`top-right`||t===`top-left`||t===`top-center`?`20px`:t===`center`?`50%`:null,right:(t===`top-right`||t===`bottom-right`)&&`20px`,bottom:(t===`bottom-left`||t===`bottom-right`||t===`bottom-center`)&&`20px`,left:t===`top-left`||t===`bottom-left`?`20px`:t===`center`||t===`top-center`||t===`bottom-center`?`50%`:null}}}}),be={name:`exclamation-triangle`,meta:{tags:[`exclamation-triangle`,`warning`,`alert`,`danger`,`caution`]},svg:{xmlns:`http://www.w3.org/2000/svg`,width:20,height:20,viewBox:`0 0 20 20`,fill:`none`},nodes:[[`path`,{d:`M10 2.25C10.2691 2.25005 10.5179 2.39429 10.6514 2.62793L18.6514 16.6279C18.7839 16.8599 18.7825 17.1448 18.6485 17.376C18.5143 17.6072 18.2673 17.75 18 17.75H2C1.73266 17.75 1.48576 17.6072 1.35156 17.376C1.21753 17.1448 1.21609 16.86 1.34863 16.6279L9.34864 2.62793C9.48218 2.39428 9.73089 2.25 10 2.25ZM3.29297 16.25H16.7071L10 4.51172L3.29297 16.25ZM10 13.25C10.4142 13.2501 10.75 13.5858 10.75 14V14.5C10.75 14.9142 10.4142 15.2499 10 15.25C9.5858 15.25 9.25001 14.9142 9.25001 14.5V14C9.25001 13.5858 9.5858 13.25 10 13.25ZM10 7.25C10.4142 7.25007 10.75 7.58583 10.75 8V11.5C10.75 11.9142 10.4142 12.2499 10 12.25C9.5858 12.25 9.25001 11.9142 9.25001 11.5V8C9.25001 7.58579 9.5858 7.25 10 7.25Z`,fill:`currentColor`,key:`dk1648`}]]},W=T({name:`ExclamationTriangle`,inheritAttrs:!1,__name:`exclamation-triangle`,setup(e){let{Icon:t}=k(be);return(e,n)=>(l(),m(a(t),x(w(e.$attrs)),null,16))}}),xe={name:`info-circle`,meta:{tags:[`info-circle`,`information`,`help`,`details`]},svg:{xmlns:`http://www.w3.org/2000/svg`,width:20,height:20,viewBox:`0 0 20 20`,fill:`none`},nodes:[[`path`,{d:`M10 1C14.9706 1 19 5.02944 19 10C19 14.9706 14.9706 19 10 19C5.02944 19 1 14.9706 1 10C1 5.02944 5.02944 1 10 1ZM10 2.5C5.85786 2.5 2.5 5.85786 2.5 10C2.5 14.1421 5.85786 17.5 10 17.5C14.1421 17.5 17.5 14.1421 17.5 10C17.5 5.85786 14.1421 2.5 10 2.5ZM10 8.25C10.4142 8.25 10.75 8.58579 10.75 9V14C10.75 14.4142 10.4142 14.75 10 14.75C9.58579 14.75 9.25 14.4142 9.25 14V9C9.25 8.58579 9.58579 8.25 10 8.25ZM10 5.25C10.4142 5.25 10.75 5.58579 10.75 6V6.5C10.75 6.91421 10.4142 7.25 10 7.25C9.58579 7.25 9.25 6.91421 9.25 6.5V6C9.25 5.58579 9.58579 5.25 10 5.25Z`,fill:`currentColor`,key:`l9ro38`}]]},G=T({name:`InfoCircle`,inheritAttrs:!1,__name:`info-circle`,setup(e){let{Icon:t}=k(xe);return(e,n)=>(l(),m(a(t),x(w(e.$attrs)),null,16))}}),Se={name:`times-circle`,meta:{tags:[`times-circle`,`close`,`cancel`,`delete`,`times`]},svg:{xmlns:`http://www.w3.org/2000/svg`,width:20,height:20,viewBox:`0 0 20 20`,fill:`none`},nodes:[[`path`,{d:`M10 1C14.9706 1 19 5.02944 19 10C19 14.9706 14.9706 19 10 19C5.02944 19 1 14.9706 1 10C1 5.02944 5.02944 1 10 1ZM10 2.5C5.85786 2.5 2.5 5.85786 2.5 10C2.5 14.1421 5.85786 17.5 10 17.5C14.1421 17.5 17.5 14.1421 17.5 10C17.5 5.85786 14.1421 2.5 10 2.5ZM12.4697 6.46973C12.7626 6.17683 13.2374 6.17683 13.5303 6.46973C13.8232 6.76262 13.8232 7.23738 13.5303 7.53027L11.0605 10L13.5303 12.4697C13.8232 12.7626 13.8232 13.2374 13.5303 13.5303C13.2374 13.8232 12.7626 13.8232 12.4697 13.5303L10 11.0605L7.53027 13.5303C7.23738 13.8232 6.76262 13.8232 6.46973 13.5303C6.17683 13.2374 6.17683 12.7626 6.46973 12.4697L8.93945 10L6.46973 7.53027C6.17683 7.23738 6.17683 6.76262 6.46973 6.46973C6.76262 6.17683 7.23738 6.17683 7.53027 6.46973L10 8.93945L12.4697 6.46973Z`,fill:`currentColor`,key:`8rdmue`}]]},K=T({name:`TimesCircle`,inheritAttrs:!1,__name:`times-circle`,setup(e){let{Icon:t}=k(Se);return(e,n)=>(l(),m(a(t),x(w(e.$attrs)),null,16))}}),Ce={name:`BaseToast`,extends:A,props:{group:{type:String,default:null},position:{type:String,default:`top-right`},mode:{type:String,default:`stacked`},gap:{type:Number,default:12},limit:{type:Number,default:3},autoZIndex:{type:Boolean,default:!0},baseZIndex:{type:Number,default:0},breakpoints:{type:Object,default:null},closeIcon:{type:String,default:void 0},infoIcon:{type:String,default:void 0},warnIcon:{type:String,default:void 0},errorIcon:{type:String,default:void 0},successIcon:{type:String,default:void 0},secondaryIcon:{type:String,default:void 0},contrastIcon:{type:String,default:void 0},closeButtonProps:{type:null,default:null},onMouseEnter:{type:Function,default:void 0},onMouseLeave:{type:Function,default:void 0},onClick:{type:Function,default:void 0}},style:ye,provide:function(){return{$pcToast:this,$parentInstance:this}}};function q(e){"@babel/helpers - typeof";return q=typeof Symbol==`function`&&typeof Symbol.iterator==`symbol`?function(e){return typeof e}:function(e){return e&&typeof Symbol==`function`&&e.constructor===Symbol&&e!==Symbol.prototype?`symbol`:typeof e},q(e)}function we(e,t,n){return(t=Te(t))in e?Object.defineProperty(e,t,{value:n,enumerable:!0,configurable:!0,writable:!0}):e[t]=n,e}function Te(e){var t=Ee(e,`string`);return q(t)==`symbol`?t:t+``}function Ee(e,t){if(q(e)!=`object`||!e)return e;var n=e[Symbol.toPrimitive];if(n!==void 0){var r=n.call(e,t);if(q(r)!=`object`)return r;throw TypeError(`@@toPrimitive must return a primitive value.`)}return(t===`string`?String:Number)(e)}var De=50,Oe=.11,J=500,Y={name:`ToastMessage`,hostName:`Toast`,extends:A,inject:[`$pcToast`],emits:[`close`],closeTimeout:null,closeRaf:null,remainingTime:0,timerStartTime:0,pointerStartPosition:null,swipeStartTime:0,props:{message:{type:null,default:null},templates:{type:Object,default:null},closeIcon:{type:String,default:null},infoIcon:{type:String,default:null},warnIcon:{type:String,default:null},errorIcon:{type:String,default:null},successIcon:{type:String,default:null},secondaryIcon:{type:String,default:null},contrastIcon:{type:String,default:null},closeButtonProps:{type:null,default:null},onMouseEnter:{type:Function,default:void 0},onMouseLeave:{type:Function,default:void 0},onClick:{type:Function,default:void 0},index:{type:Number,default:0}},data:function(){return{isMounted:!1,measuredHeight:0,removed:!1,offsetBeforeRemove:0,swiping:!1,isSwiped:!1,swipeOut:!1,swipeDirection:null,swipeOutDirection:null,swipeAmountX:0,swipeAmountY:0}},watch:{shouldPauseTimer:function(e){this.removed||(e?this.pauseTimer():this.startTimer())}},mounted:function(){var e,t;this.measureHeight(),this.isMounted=!0,(e=this.$pcToast)==null||(t=e.onEnter)==null||t.call(e),this.shouldPauseTimer||this.startTimer()},beforeUnmount:function(){var e,t;this.clearCloseTimeout(),(e=this.$pcToast)==null||(t=e.onLeave)==null||t.call(e)},unmounted:function(){if(this.removed){var e,t;(e=this.$pcToast)==null||(t=e.onAfterLeave)==null||t.call(e)}},methods:{measureHeight:function(){var e,t,n=this.$refs.messageEl;if(n){var r=n.style.height;n.style.height=`auto`;var i=n.getBoundingClientRect().height;n.style.height=r,this.measuredHeight=i,(e=this.$pcToast)==null||(t=e.onItemHeightChange)==null||t.call(e,{index:this.index,height:i})}},startTimer:function(){var e=this;if(this.clearCloseTimeout(),!this.message.sticky){if(!this.remainingTime||this.remainingTime<=0){if(!this.message.life)return;this.remainingTime=this.message.life}this.timerStartTime=Date.now(),this.closeTimeout=setTimeout(function(){e.onMessageRemoveFocus(),e.closeStack()},this.remainingTime)}},pauseTimer:function(){if(this.timerStartTime>0&&this.closeTimeout){var e=Date.now()-this.timerStartTime;this.remainingTime=Math.max(0,this.remainingTime-e)}this.clearCloseTimeout()},markRemoved:function(){var e,t;this.offsetBeforeRemove=this.offset,this.removed=!0,(e=this.$pcToast)==null||(t=e.onItemHeightChange)==null||t.call(e,{index:this.index,height:0,removed:!0})},isDismissible:function(){return this.message?.closable!==!1},onPointerDown:function(e){if(e.button===0&&this.isDismissible()){this.swipeStartTime=Date.now(),this.offsetBeforeRemove=this.offset;try{e.target.setPointerCapture(e.pointerId)}catch{}this.swiping=!0,this.pointerStartPosition={x:e.clientX,y:e.clientY}}},onPointerMove:function(e){if(!(!this.pointerStartPosition||!this.isDismissible())&&!((window.getSelection()?.toString().length??0)>0)){var t=e.clientY-this.pointerStartPosition.y,n=e.clientX-this.pointerStartPosition.x,r=Math.abs(n)>1||Math.abs(t)>1,i=(this.$pcToast?.position??`top-right`).split(`-`),a=i[0],o=i[1];!this.swipeDirection&&r&&(this.swipeDirection=Math.abs(n)>Math.abs(t)?`x`:`y`);var s=0,c=0;this.swipeDirection===`x`?s=o===`left`&&n<0||o===`right`&&n>0?n:this.applyDampening(n):this.swipeDirection===`y`&&(c=a===`top`&&t<0||a===`bottom`&&t>0?t:this.applyDampening(t)),(Math.abs(s)>0||Math.abs(c)>0)&&(this.isSwiped=!0),this.swipeAmountX=s,this.swipeAmountY=c}},onPointerUp:function(){if(!(this.swipeOut||!this.isDismissible())){this.swiping=!1,this.pointerStartPosition=null;var e=this.swipeDirection===`x`?this.swipeAmountX:this.swipeAmountY,t=Date.now()-(this.swipeStartTime||Date.now()),n=t>0?Math.abs(e)/t:0;if(Math.abs(e)>=De||n>Oe){this.offsetBeforeRemove=this.offset,this.swipeOutDirection=this.swipeDirection===`x`?this.swipeAmountX>0?`right`:`left`:this.swipeAmountY>0?`down`:`up`,this.swipeOut=!0,this.markRemoved(),this.scheduleSwipeOutClose();return}this.swipeAmountX=0,this.swipeAmountY=0,this.isSwiped=!1,this.swipeDirection=null}},onDragEnd:function(){this.swiping=!1,this.swipeDirection=null,this.pointerStartPosition=null},applyDampening:function(e){var t=e*(1/(1.5+Math.abs(e)/20));return Math.abs(t)<Math.abs(e)?t:e},scheduleSwipeOutClose:function(){var e=this;this.clearCloseTimeout(),this.closeTimeout=setTimeout(function(){e.close({message:e.message,type:`close`})},J)},scheduleClose:function(e){var t=this;this.clearCloseTimeout(),this.closeRaf=requestAnimationFrame(function(){t.closeRaf=null;var n=t.$refs.messageEl,r=n?(parseFloat(getComputedStyle(n).transitionDuration)||0)*1e3:0;t.closeTimeout=setTimeout(function(){t.close({message:t.message,type:e})},r||J)})},closeStack:function(){this.markRemoved(),this.scheduleClose(`life-end`)},close:function(e){this.$emit(`close`,e)},onCloseClick:function(){this.clearCloseTimeout(),this.onMessageRemoveFocus(),this.markRemoved(),this.scheduleClose(`close`)},onMessageRemoveFocus:function(){var e=this.$refs.messageEl;if(e){var t=document.activeElement;if(e.contains(t)){var n=`[data-pc-section="closebutton"]:not([tabindex="-1"])`,r=e.nextElementSibling?.querySelector(n),i=e.previousElementSibling?.querySelector(n);requestAnimationFrame(function(){r?r.focus({preventScroll:!0}):i&&i.focus({preventScroll:!0})})}}},clearCloseTimeout:function(){this.closeTimeout&&=(clearTimeout(this.closeTimeout),null),this.closeRaf&&=(cancelAnimationFrame(this.closeRaf),null)},onMessageClick:function(e){var t;(t=this.onClick)==null||t.call(this,{originalEvent:e,message:this.message})},onMessageMouseEnter:function(e){var t;(t=this.onMouseEnter)==null||t.call(this,{originalEvent:e,message:this.message})},onMessageMouseLeave:function(e){var t;(t=this.onMouseLeave)==null||t.call(this,{originalEvent:e,message:this.message})},resolveIcon:function(e){return L(e)?e:r(e)},isComponentIcon:function(e){return!!e&&!L(e)}},computed:{isExpanded:function(){return this.$pcToast?.isExpanded??!1},toastCount:function(){var e;return((e=this.$pcToast)==null||(e=e.messages)==null?void 0:e.length)??0},isVisible:function(){var e,t;return((e=this.$pcToast)==null||(t=e.getIsVisible)==null?void 0:t.call(e,this.index))??!1},stackExpanded:function(){return this.$pcToast?.expanded??!1},visibleIndex:function(){var e,t;return((e=this.$pcToast)==null||(t=e.getVisibleIndex)==null?void 0:t.call(e,this.index))??0},offset:function(){var e,t;return((e=this.$pcToast)==null||(t=e.getOffset)==null?void 0:t.call(e,this.index))??0},isInteracting:function(){return this.$pcToast?.isInteracting??!1},shouldPauseTimer:function(){return this.stackExpanded||this.isInteracting||this.swiping},isAriaHidden:function(){return!this.isVisible&&!this.removed?`true`:null},isTabbable:function(){return!this.removed&&this.isVisible},stackStyles:function(){return{"--px-toast-index":this.removed?this.index:this.visibleIndex,"--px-toast-z-index":this.toastCount-this.visibleIndex,"--px-initial-height":this.measuredHeight?`${this.measuredHeight}px`:void 0,"--px-toast-offset":`${this.removed?this.offsetBeforeRemove:this.offset}px`,"--px-swipe-amount-x":`${this.swipeAmountX}px`,"--px-swipe-amount-y":`${this.swipeAmountY}px`,"z-index":this.toastCount-this.visibleIndex}},iconComponent:function(){return{info:this.infoIcon?`span`:G,success:this.successIcon?`span`:M,warn:this.warnIcon?`span`:W,error:this.errorIcon?`span`:K,secondary:this.secondaryIcon?`span`:G,contrast:this.contrastIcon?`span`:G}[this.message.severity]},closeAriaLabel:function(){return this.$primevue.config.locale.aria?this.$primevue.config.locale.aria.close:void 0},dataP:function(){return D(we({},this.message.severity,this.message.severity))}},components:{Times:ue,InfoCircle:G,Check:M,ExclamationTriangle:W,TimesCircle:K},directives:{ripple:re}};function X(e){"@babel/helpers - typeof";return X=typeof Symbol==`function`&&typeof Symbol.iterator==`symbol`?function(e){return typeof e}:function(e){return e&&typeof Symbol==`function`&&e.constructor===Symbol&&e!==Symbol.prototype?`symbol`:typeof e},X(e)}function ke(e,t){var n=Object.keys(e);if(Object.getOwnPropertySymbols){var r=Object.getOwnPropertySymbols(e);t&&(r=r.filter(function(t){return Object.getOwnPropertyDescriptor(e,t).enumerable})),n.push.apply(n,r)}return n}function Ae(e){for(var t=1;t<arguments.length;t++){var n=arguments[t]==null?{}:arguments[t];t%2?ke(Object(n),!0).forEach(function(t){je(e,t,n[t])}):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(n)):ke(Object(n)).forEach(function(t){Object.defineProperty(e,t,Object.getOwnPropertyDescriptor(n,t))})}return e}function je(e,t,n){return(t=Me(t))in e?Object.defineProperty(e,t,{value:n,enumerable:!0,configurable:!0,writable:!0}):e[t]=n,e}function Me(e){var t=Ne(e,`string`);return X(t)==`symbol`?t:t+``}function Ne(e,t){if(X(e)!=`object`||!e)return e;var n=e[Symbol.toPrimitive];if(n!==void 0){var r=n.call(e,t);if(X(r)!=`object`)return r;throw TypeError(`@@toPrimitive must return a primitive value.`)}return(t===`string`?String:Number)(e)}var Pe=[`aria-hidden`,`data-p`,`data-id`,`data-index`,`data-mounted`,`data-removed`,`data-front`,`data-expanded`,`data-visible`,`data-swiping`,`data-swiped`,`data-swipe-out`,`data-swipe-direction`,`data-dismissible`],Fe=[`data-p`],Ie=[`data-p`],Le=[`data-p`],Re=[`aria-label`,`tabindex`,`data-p`];function ze(e,t,n,r,i,a){var o=c(`ripple`);return l(),_(`div`,C({ref:`messageEl`,class:[e.cx(`message`),n.message.styleClass],role:`alert`,"aria-live":`assertive`,"aria-atomic":`true`,"aria-hidden":a.isAriaHidden,"data-p":a.dataP,"data-id":n.message?.id,"data-index":n.index,"data-stack":``,"data-mounted":i.isMounted?``:void 0,"data-removed":i.removed?``:void 0,"data-front":a.visibleIndex===0?``:void 0,"data-expanded":a.isExpanded?``:void 0,"data-visible":a.isVisible?``:void 0,"data-swiping":i.swiping?``:void 0,"data-swiped":i.isSwiped?``:void 0,"data-swipe-out":i.swipeOut?``:void 0,"data-swipe-direction":i.swipeOutDirection?i.swipeOutDirection:void 0,"data-dismissible":String(a.isDismissible()),style:a.stackStyles},e.ptm(`message`),{onClick:t[1]||=function(){return a.onMessageClick&&a.onMessageClick.apply(a,arguments)},onMouseenter:t[2]||=function(){return a.onMessageMouseEnter&&a.onMessageMouseEnter.apply(a,arguments)},onMouseleave:t[3]||=function(){return a.onMessageMouseLeave&&a.onMessageMouseLeave.apply(a,arguments)},onPointerdown:t[4]||=function(){return a.onPointerDown&&a.onPointerDown.apply(a,arguments)},onPointermove:t[5]||=function(){return a.onPointerMove&&a.onPointerMove.apply(a,arguments)},onPointerup:t[6]||=function(){return a.onPointerUp&&a.onPointerUp.apply(a,arguments)},onDragend:t[7]||=function(){return a.onDragEnd&&a.onDragEnd.apply(a,arguments)}}),[n.templates.container?(l(),m(u(n.templates.container),{key:0,message:n.message,closeCallback:a.onCloseClick},null,8,[`message`,`closeCallback`])):(l(),_(`div`,C({key:1,class:[e.cx(`messageContent`),n.message.contentStyleClass]},e.ptm(`messageContent`)),[n.templates.message?(l(),m(u(n.templates.message),{key:1,message:n.message},null,8,[`message`])):(l(),_(b,{key:0},[n.templates.messageicon?(l(),m(u(n.templates.messageicon),C({key:0,message:n.message,class:e.cx(`messageIcon`)},e.ptm(`messageIcon`)),null,16,[`message`,`class`])):a.isComponentIcon(n.message.icon)?(l(),m(u(a.resolveIcon(n.message.icon)),C({key:1,class:e.cx(`messageIcon`)},e.ptm(`messageIcon`)),null,16,[`class`])):n.message.icon?(l(),_(`span`,C({key:2,class:[e.cx(`messageIcon`),n.message.icon]},e.ptm(`messageIcon`)),null,16)):a.iconComponent?(l(),m(u(a.iconComponent),C({key:3,class:e.cx(`messageIcon`)},e.ptm(`messageIcon`)),null,16,[`class`])):y(``,!0),p(`div`,C({class:e.cx(`messageText`),"data-p":a.dataP},e.ptm(`messageText`)),[p(`span`,C({class:e.cx(`summary`),"data-p":a.dataP},e.ptm(`summary`)),g(n.message.summary),17,Ie),n.message.detail?(l(),_(`div`,C({key:0,class:e.cx(`detail`),"data-p":a.dataP},e.ptm(`detail`)),g(n.message.detail),17,Le)):y(``,!0)],16,Fe)],64)),n.message.closable===!1?y(``,!0):(l(),_(`div`,x(C({key:2},e.ptm(`buttonContainer`))),[E((l(),_(`button`,C({class:e.cx(`closeButton`),type:`button`,"aria-label":a.closeAriaLabel,tabindex:a.isTabbable?null:-1,onClick:t[0]||=function(){return a.onCloseClick&&a.onCloseClick.apply(a,arguments)},"data-p":a.dataP},Ae(Ae({},n.closeButtonProps),e.ptm(`closeButton`))),[(l(),m(u(n.templates.closeicon||`Times`),C({class:[e.cx(`closeIcon`),n.closeIcon]},e.ptm(`closeIcon`)),null,16,[`class`]))],16,Re)),[[o]])],16))],16))],16,Pe)}Y.render=ze;function Z(e){"@babel/helpers - typeof";return Z=typeof Symbol==`function`&&typeof Symbol.iterator==`symbol`?function(e){return typeof e}:function(e){return e&&typeof Symbol==`function`&&e.constructor===Symbol&&e!==Symbol.prototype?`symbol`:typeof e},Z(e)}function Be(e,t){var n=Object.keys(e);if(Object.getOwnPropertySymbols){var r=Object.getOwnPropertySymbols(e);t&&(r=r.filter(function(t){return Object.getOwnPropertyDescriptor(e,t).enumerable})),n.push.apply(n,r)}return n}function Ve(e){for(var t=1;t<arguments.length;t++){var n=arguments[t]==null?{}:arguments[t];t%2?Be(Object(n),!0).forEach(function(t){He(e,t,n[t])}):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(n)):Be(Object(n)).forEach(function(t){Object.defineProperty(e,t,Object.getOwnPropertyDescriptor(n,t))})}return e}function He(e,t,n){return(t=Ue(t))in e?Object.defineProperty(e,t,{value:n,enumerable:!0,configurable:!0,writable:!0}):e[t]=n,e}function Ue(e){var t=We(e,`string`);return Z(t)==`symbol`?t:t+``}function We(e,t){if(Z(e)!=`object`||!e)return e;var n=e[Symbol.toPrimitive];if(n!==void 0){var r=n.call(e,t);if(Z(r)!=`object`)return r;throw TypeError(`@@toPrimitive must return a primitive value.`)}return(t===`string`?String:Number)(e)}function Q(e){return Je(e)||qe(e)||Ke(e)||Ge()}function Ge(){throw TypeError(`Invalid attempt to spread non-iterable instance.
In order to be iterable, non-array objects must have a [Symbol.iterator]() method.`)}function Ke(e,t){if(e){if(typeof e==`string`)return $(e,t);var n={}.toString.call(e).slice(8,-1);return n===`Object`&&e.constructor&&(n=e.constructor.name),n===`Map`||n===`Set`?Array.from(e):n===`Arguments`||/^(?:Ui|I)nt(?:8|16|32)(?:Clamped)?Array$/.test(n)?$(e,t):void 0}}function qe(e){if(typeof Symbol<`u`&&e[Symbol.iterator]!=null||e[`@@iterator`]!=null)return Array.from(e)}function Je(e){if(Array.isArray(e))return $(e)}function $(e,t){(t==null||t>e.length)&&(t=e.length);for(var n=0,r=Array(t);n<t;n++)r[n]=e[n];return r}var Ye=0,Xe={name:`Toast`,extends:Ce,inheritAttrs:!1,emits:[`close`,`life-end`],data:function(){return{messages:[],expanded:!1,removingCount:0,heights:[],isInteracting:!1}},styleElement:null,zIndexClearTimeout:null,mounted:function(){I.on(`add`,this.onAdd),I.on(`remove`,this.onRemove),I.on(`remove-group`,this.onRemoveGroup),I.on(`remove-all-groups`,this.onRemoveAllGroups),this.breakpoints&&this.createStyle()},beforeUnmount:function(){this.destroyStyle(),this.zIndexClearTimeout&&=(clearTimeout(this.zIndexClearTimeout),null),this.$refs.container&&this.autoZIndex&&N.clear(this.$refs.container),I.off(`add`,this.onAdd),I.off(`remove`,this.onRemove),I.off(`remove-group`,this.onRemoveGroup),I.off(`remove-all-groups`,this.onRemoveAllGroups)},methods:{add:function(e){e.id??=Ye++,this.messages=[].concat(Q(this.messages),[e])},remove:function(e){var t=this.messages.findIndex(function(t){return t.id===e.message.id});t!==-1&&(this.messages.splice(t,1),this.heights=this.heights.filter(function(e){return e.index!==t}).map(function(e){return e.index>t?Ve(Ve({},e),{},{index:e.index-1}):e}),this.messages.length<=1&&(this.expanded=!1),this.$emit(e.type,{message:e.message}))},onAdd:function(e){this.group==e.group&&this.add(e)},onRemove:function(e){this.remove({message:e,type:`close`})},onRemoveGroup:function(e){this.group===e&&(this.messages=[],this.heights=[],this.removingCount=0,this.expanded=!1,this.isInteracting=!1)},onRemoveAllGroups:function(){var e=this,t=this.messages;this.messages=[],this.heights=[],this.removingCount=0,this.expanded=!1,this.isInteracting=!1,t.forEach(function(t){return e.$emit(`close`,{message:t})})},onEnter:function(){this.autoZIndex&&this.$refs.container&&this.$refs.container.style.zIndex===``&&N.set(`modal`,this.$refs.container,this.baseZIndex||this.$primevue.config.zIndex.modal)},onLeave:function(){var e=this;this.$refs.container&&this.autoZIndex&&le(this.messages)&&(this.zIndexClearTimeout&&clearTimeout(this.zIndexClearTimeout),this.zIndexClearTimeout=setTimeout(function(){N.clear(e.$refs.container),e.zIndexClearTimeout=null},200))},onAfterLeave:function(){this.removingCount=Math.max(0,this.removingCount-1)},onContainerMouseEnter:function(){this.expanded=!0},onContainerMouseLeave:function(e){this.isInteracting||this.isPointerOrFocusInside(e.relatedTarget)||(this.expanded=!1)},onContainerFocusIn:function(){this.expanded=!0},onContainerFocusOut:function(e){this.isInteracting||this.isPointerOrFocusInside(e.relatedTarget)||(this.expanded=!1)},onContainerPointerDown:function(e){var t=e.target;t instanceof HTMLElement&&t.closest(`[data-dismissible="false"]`)||(this.isInteracting=!0)},onContainerPointerUp:function(){this.isInteracting=!1},isPointerOrFocusInside:function(e){var t=this.$refs.container;return!!(e&&t&&t.contains(e))},onItemHeightChange:function(e){if(e.removed){this.heights=this.heights.filter(function(t){return t.index!==e.index}),this.removingCount+=1;return}var t=this.heights.findIndex(function(t){return t.index===e.index});if(t>=0){var n=Q(this.heights);n[t]={index:e.index,height:e.height},this.heights=n}else this.heights=[].concat(Q(this.heights),[{index:e.index,height:e.height}]).sort(function(e,t){return e.index-t.index})},getVisibleIndex:function(e){return this.visibleIndexMap.get(e)??this.messages.length-1-e},getOffset:function(e){var t=this.visibleIndexMap.get(e)??0;return this.offsets[t]??0},getIsVisible:function(e){return this.visibleDomIndices.has(e)},createStyle:function(){if(!this.styleElement&&!this.isUnstyled){var e;this.styleElement=document.createElement(`style`),this.styleElement.type=`text/css`,oe(this.styleElement,`nonce`,(e=this.$primevue)==null||(e=e.config)==null||(e=e.csp)==null?void 0:e.nonce),document.head.appendChild(this.styleElement);var t=``;for(var n in this.breakpoints){var r=``;for(var i in this.breakpoints[n])r+=i+`:`+this.breakpoints[n][i]+`!important;`;t+=`
                        @media screen and (max-width: ${n}) {
                            .p-toast[${this.$attrSelector}] {
                                ${r}
                            }
                        }
                    `}this.styleElement.innerHTML=t}},destroyStyle:function(){this.styleElement&&=(document.head.removeChild(this.styleElement),null)}},computed:{isExpanded:function(){return this.mode===`expanded`||this.expanded},sortedHeights:function(){return Q(this.heights).sort(function(e,t){return t.index-e.index})},frontToastHeight:function(){return this.sortedHeights[0]?.height??0},offsets:function(){for(var e=this.sortedHeights,t=[0],n=1;n<e.length;n++)t[n]=t[n-1]+e[n-1].height;return t},visibleIndexMap:function(){var e=new Map;return this.sortedHeights.forEach(function(t,n){return e.set(t.index,n)}),e},visibleDomIndices:function(){return new Set(this.sortedHeights.slice(0,this.limit).map(function(e){return e.index}))},raiseFactor:function(){return(this.position||``).startsWith(`bottom`)?-1:1},hostDataExpanded:function(){return this.isExpanded?``:null},containerStyle:function(){return[this.sx(`root`,!0,{position:this.position}),{"--px-gap":`${this.gap}px`,"--px-front-toast-height":`${this.frontToastHeight}px`,"--px-raise-factor":this.raiseFactor}]},dataP:function(){return D(He({},this.position,this.position))}},components:{ToastMessage:Y,Portal:ie}},Ze=[`data-p`,`data-position`,`data-expanded`];function Qe(t,n,r,i,a,o){var c=s(`ToastMessage`),u=s(`Portal`);return l(),m(u,null,{default:d(function(){return[p(`div`,C({ref:`container`,class:t.cx(`root`),style:o.containerStyle,"data-p":o.dataP,"data-position":t.position,"data-expanded":o.hostDataExpanded},t.ptmi(`root`),{onMouseenter:n[1]||=function(){return o.onContainerMouseEnter&&o.onContainerMouseEnter.apply(o,arguments)},onMouseleave:n[2]||=function(){return o.onContainerMouseLeave&&o.onContainerMouseLeave.apply(o,arguments)},onFocusin:n[3]||=function(){return o.onContainerFocusIn&&o.onContainerFocusIn.apply(o,arguments)},onFocusout:n[4]||=function(){return o.onContainerFocusOut&&o.onContainerFocusOut.apply(o,arguments)},onPointerdown:n[5]||=function(){return o.onContainerPointerDown&&o.onContainerPointerDown.apply(o,arguments)},onPointerup:n[6]||=function(){return o.onContainerPointerUp&&o.onContainerPointerUp.apply(o,arguments)}}),[(l(!0),_(b,null,e(a.messages,function(e,r){return l(),m(c,{key:e.id,index:r,message:e,templates:t.$slots,closeIcon:t.closeIcon,infoIcon:t.infoIcon,warnIcon:t.warnIcon,errorIcon:t.errorIcon,successIcon:t.successIcon,secondaryIcon:t.secondaryIcon,contrastIcon:t.contrastIcon,closeButtonProps:t.closeButtonProps,onMouseEnter:t.onMouseEnter,onMouseLeave:t.onMouseLeave,onClick:t.onClick,unstyled:t.unstyled,onClose:n[0]||=function(e){return o.remove(e)},pt:t.pt},null,8,[`index`,`message`,`templates`,`closeIcon`,`infoIcon`,`warnIcon`,`errorIcon`,`successIcon`,`secondaryIcon`,`contrastIcon`,`closeButtonProps`,`onMouseEnter`,`onMouseLeave`,`onClick`,`unstyled`,`pt`])}),128))],16,Ze)]}),_:1})}Xe.render=Qe;var $e=h(`websocket`,()=>{let e=i(!1),t=i([]),n=null,r=null;function a(i){if(n)return;let o=`${location.protocol===`https:`?`wss:`:`ws:`}//${location.host}/api/ws`;n=new WebSocket(o),n.onopen=()=>{e.value=!0,n.send(JSON.stringify({type:`auth`,token:i}))},n.onclose=()=>{e.value=!1,n=null,r=setTimeout(()=>a(i),3e3)},n.onmessage=e=>{let n=JSON.parse(e.data);t.value.push(n),t.value.length>100&&t.value.shift()}}function o(){r&&clearTimeout(r),n&&=(n.close(),null),e.value=!1}function s(e){n&&n.readyState===WebSocket.OPEN&&n.send(JSON.stringify({type:`mark_read`,taskId:e}))}return{connected:e,events:t,connect:a,disconnect:o,markAsRead:s}}),et={class:`filter-bar`},tt=P({__name:`FilterBar`,setup(e){let t=z(),r=i(``),o=i(null),s=i(null),c=[{label:`Входящие`,value:`Входящие`},{label:`ТРК`,value:`ТРК`},{label:`Отель`,value:`Отель`},{label:`Лидер Спорт`,value:`Лидер Спорт`},{label:`Театр`,value:`Театр`},{label:`Мебельный центр`,value:`Мебельный центр`},{label:`Кафе`,value:`Кафе`},{label:`Ледовая арена`,value:`Ледовая арена`},{label:`Корпоративные сайты`,value:`Корпоративные сайты`}];n(()=>{r.value=t.filters.search||``,o.value=t.filters.project||null,s.value=t.filters.status||null});let u;function d(){clearTimeout(u),u=setTimeout(()=>{t.setFilter(`search`,r.value)},300)}function f(e,n){t.setFilter(e,n||``)}return(e,t)=>(l(),_(`div`,et,[S(a(de),{modelValue:r.value,"onUpdate:modelValue":t[0]||=e=>r.value=e,placeholder:`Поиск...`,onInput:d,class:`search-input`},null,8,[`modelValue`]),S(a(ae),{modelValue:o.value,"onUpdate:modelValue":t[1]||=e=>o.value=e,options:c,optionLabel:`label`,optionValue:`value`,placeholder:`Проект`,onChange:t[2]||=e=>f(`project`,e.value),showClear:``},null,8,[`modelValue`])]))}},[[`__scopeId`,`data-v-43512834`]]),nt={class:`subject-cell`},rt=P({__name:`TaskTable`,setup(e){let t=z(),n=R(),r=F();function i(e){if(!e)return``;let t=new Date(e);return t.toLocaleDateString(`ru-RU`)+` `+t.toLocaleTimeString(`ru-RU`,{hour:`2-digit`,minute:`2-digit`})}function o(e){return{new:`Новая`,in_progress:`В работе`,resolved:`Решена`,closed:`Закрыта`}[e]||e}function s(e){return{new:`info`,in_progress:`warn`,resolved:`success`,closed:`secondary`}[e]||`info`}function c(e){return{urgent:`danger`,high:`warn`,medium:`info`,low:`success`}[e]||`info`}function u(e){return e.unread_comments>0?`task-unread`:``}function h(e){n.push({path:`/tasks/${e.data.id}`,query:{tab:r.query.tab}})}function _(e){t.filters.page=e.page+1,t.fetchTasks()}return(e,n)=>(l(),m(a(pe),{value:a(t).tasks,loading:a(t).loading,paginator:``,rows:50,totalRecords:a(t).total,onPage:_,lazy:``,stripedRows:``,rowClass:u,onRowClick:h},{default:d(()=>[S(a(V),{field:`id`,header:`ID`,style:{width:`80px`}}),S(a(V),{field:`created_at`,header:`Дата`,style:{width:`150px`}},{body:d(({data:e})=>[f(g(i(e.created_at)),1)]),_:1}),S(a(V),{field:`from_email`,header:`От кого`,style:{width:`200px`}}),S(a(V),{field:`subject`,header:`Тема`},{body:d(({data:e})=>[p(`div`,nt,[e.unread_comments>0?(l(),m(a(O),{key:0,value:e.unread_comments,severity:`info`,size:`small`,class:`unread-badge`},null,8,[`value`])):y(``,!0),p(`span`,null,g(e.subject),1)])]),_:1}),S(a(V),{field:`type`,header:`Тип`,style:{width:`100px`}},{body:d(({data:e})=>[e.type?(l(),m(a(B),{key:0,value:e.type},null,8,[`value`])):y(``,!0)]),_:1}),S(a(V),{field:`priority`,header:`Приоритет`,style:{width:`100px`}},{body:d(({data:e})=>[S(a(B),{value:e.priority,severity:c(e.priority)},null,8,[`value`,`severity`])]),_:1}),S(a(V),{field:`project`,header:`Проект`,style:{width:`150px`}}),S(a(V),{field:`status`,header:`Статус`,style:{width:`120px`}},{body:d(({data:e})=>[S(a(B),{value:o(e.status),severity:s(e.status)},null,8,[`value`,`severity`])]),_:1}),S(a(V),{field:`assignee`,header:`Исполнитель`,style:{width:`120px`}})]),_:1},8,[`value`,`loading`,`totalRecords`]))}},[[`__scopeId`,`data-v-8be317be`]]),it={class:`tab-bar`},at=[`onClick`],ot=P({__name:`TabBar`,props:{tabs:{type:Array,required:!0},activeTab:{type:String,required:!0}},emits:[`select`],setup(t){return(n,r)=>(l(),_(`div`,it,[(l(!0),_(b,null,e(t.tabs,e=>(l(),_(`button`,{key:e.key,class:v({active:t.activeTab===e.key}),onClick:t=>n.$emit(`select`,e.key)},[f(g(e.label)+` `,1),e.count>0?(l(),m(a(O),{key:0,value:e.count,severity:`info`,size:`small`},null,8,[`value`])):y(``,!0)],10,at))),128))]))}},[[`__scopeId`,`data-v-551a7ff4`]]),st={class:`dashboard`},ct={class:`dashboard-header`},lt={class:`header-right`},ut={class:`task-count`},dt={class:`inbox-count`},ft={class:`dashboard-content`},pt=P({__name:`DashboardView`,setup(e){let r=he(),s=R(),c=F(),u=se(),d=te(),f=z(),h=$e(),y=me(),x=i(`active`),C=i(0),w=ne(()=>[{key:`inbox`,label:`Лента`,count:0},{key:`active`,label:`Активные`,count:0},{key:`backlog`,label:`Бэклог`,count:0},{key:`completed`,label:`Выполненные`,count:0},{key:`closed`,label:`Закрытые`,count:0}]),T={active:[`new`,`in_progress`],backlog:[`backlog`],completed:[`completed`],closed:[`closed`]};n(()=>{h.connect(d.token);let e=c.query.tab,t=localStorage.getItem(`mailbridge_active_tab`),n=`active`;e&&(T[e]||e===`inbox`)?n=e:t&&(T[t]||t===`inbox`)&&(n=t),x.value=n,n!==`inbox`&&f.setStatuses(T[n]),E(),y.fetchUnreadCount()}),t(()=>{h.disconnect()});async function E(){try{let{data:e}=await ee.get(`/tasks`,{params:{status:[`new`,`in_progress`],page:1,per_page:1}});C.value=e.total||0}catch{C.value=0}}function D(e){x.value=e,localStorage.setItem(`mailbridge_active_tab`,e),s.replace({query:{...c.query,tab:e}}),e!==`inbox`&&f.setStatuses(T[e])}o(()=>h.events.length,()=>{let e=h.events,t=e[e.length-1];if(t)switch(t.type){case`task_created`:f.fetchTasks(),E(),u.add({severity:`info`,summary:t.message,life:5e3});break;case`task_updated`:f.fetchTasks(),E(),u.add({severity:`warn`,summary:t.message,life:5e3});break;case`inbox_created`:y.fetchItems(),y.fetchUnreadCount(),u.add({severity:`info`,summary:t.message,life:5e3});break;case`connected`:u.add({severity:`success`,summary:t.message,life:2e3})}});function re(){h.disconnect(),d.logout(),s.push(`/login`)}return(e,t)=>(l(),_(`div`,st,[S(a(Xe),{position:`top-right`}),p(`header`,ct,[t[1]||=p(`h1`,null,`Mailbridge`,-1),p(`div`,lt,[p(`span`,{class:v([`connection-status`,{connected:a(h).connected}])},g(a(h).connected?`● Онлайн`:`○ Офлайн`),3),p(`span`,ut,`Задачи: `+g(C.value),1),p(`span`,dt,`Входящие: `+g(a(y).unreadCount),1),S(a(j),{label:`Выйти`,severity:`secondary`,onClick:re}),S(a(j),{icon:a(r).isDark?`pi pi-sun`:`pi pi-moon`,text:``,onClick:t[0]||=e=>a(r).toggleTheme(),title:`Переключить тему`},null,8,[`icon`])])]),p(`main`,ft,[S(ot,{tabs:w.value,activeTab:x.value,onSelect:D},null,8,[`tabs`,`activeTab`]),x.value===`inbox`?(l(),m(fe,{key:0})):(l(),_(b,{key:1},[S(tt),S(rt)],64))])]))}},[[`__scopeId`,`data-v-446a56b8`]]);export{pt as default};