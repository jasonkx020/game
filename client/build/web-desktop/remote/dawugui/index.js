System.register("chunks:///_virtual/dawugui",["./DawuguiModule.ts","./DawuguiPushHandler.ts","./GameEntry.ts","./StandaloneEntry.ts"],(function(){return{setters:[null,null,null,null],execute:function(){}}}));

System.register("chunks:///_virtual/DawuguiModule.ts",["./rollupPluginModLoBabelHelpers.js","cc","./GameModuleRegistry.ts","./DawuguiPushHandler.ts","./protoHelpers.ts","./dawugui.ts","./SessionStore.ts"],(function(e){var n,t,o,r,u,s,a,i,c,l,d,g,p;return{setters:[function(e){n=e.asyncToGenerator,t=e.regeneratorRuntime,o=e.createForOfIteratorHelperLoose},function(e){r=e.cclegacy},function(e){u=e.GameModuleRegistry},function(e){s=e.registerDawuguiPushHandlers},function(e){a=e.decodeProto,i=e.encodeProto},function(e){c=e.PassRsp,l=e.PassReq,d=e.PlayCardsRsp,g=e.PlayCardsReq},function(e){p=e.SessionStore}],execute:function(){e("registerDawuguiModule",(function(){var e={gameId:"dawugui",registerPush:function(e){s(e.router,(function(n){e.log(n),function(e,n,t){var r,s,a=String(t),i=u.get("dawugui"),c="push";a.includes("onAlert")&&(c="onAlert");a.includes("onSettlement")&&(c="onSettlement");for(var l,d=null!=(r=null==i||null==(s=i.companionHooks)||null==s.onPublicEvent?void 0:s.onPublicEvent(c,t))?r:[],g=o(d);!(l=g()).done;){var f=l.value;p.appendCompanionHint(f.text)}}(0,0,n)}))},companionHooks:{getRulesSummary:function(){return"打乌龟是 3-5 人扑克跑牌游戏，先出完手牌获胜。注意报单与包牌。"},getStrategyTips:function(e){return"playing"===e?["观察对手出牌习惯","报单后要谨慎"]:["熟悉牌型大小优先"]},onPublicEvent:function(e,n){return"onAlert"===e?[{text:"有人报单了，注意防守！"}]:"onSettlement"===e?[{text:"本局结束啦，要不要再来一局？"}]:[]}},onPassClick:function(e){return n(t().mark((function n(){var o,r;return t().wrap((function(n){for(;;)switch(n.prev=n.next){case 0:return o=i(l,{roomId:e.roomId}),n.next=3,e.session.pitaya.request("game.dawugui.pass",o);case 3:r=n.sent,a(c,r),e.log("[action] pass sent");case 6:case"end":return n.stop()}}),n)})))()},onPlayClick:function(e){return n(t().mark((function n(){var o,r;return t().wrap((function(n){for(;;)switch(n.prev=n.next){case 0:return o=i(g,{roomId:e.roomId,cards:[1]}),n.next=3,e.session.pitaya.request("game.dawugui.playcards",o);case 3:r=n.sent,a(d,r),e.log("[action] playcards sent");case 6:case"end":return n.stop()}}),n)})))()}};u.register(e)})),r._RF.push({},"3b228JbABNLioKdh4qrZjSz","DawuguiModule",void 0),r._RF.pop()}}}));

System.register("chunks:///_virtual/DawuguiPushHandler.ts",["cc","./dawugui.ts","./common.ts","./room.ts"],(function(n){var e,t,o,u,r,l,s,c,a;return{setters:[function(n){e=n.cclegacy},function(e){t=e.DealPush,o=e.TurnNotifyPush,u=e.PlayResultPush,r=e.AlertPush,l=e.RoundInvalidPush,s=e.SettlementPush;var c={};c.DealPush=e.DealPush,c.PlayResultPush=e.PlayResultPush,c.SettlementPush=e.SettlementPush,n(c)},function(n){c=n.ErrorPush},function(n){a=n.RoomStatePush}],execute:function(){function i(n,e){var t,o;return"[push:"+n+"] action_seq="+(null!=(t=null==(o=e.header)||null==(o=o.meta)?void 0:o.actionSeq)?t:0)}n("registerDawuguiPushHandlers",(function(n,e){void 0===e&&(e=console.log);for(var d=0,f=[["onRoomState",function(n,t){return e(i("onRoomState",a.decode(t)))}],["onDeal",function(n,o){return e(i("onDeal",t.decode(o)))}],["onTurnNotify",function(n,t){return e(i("onTurnNotify",o.decode(t)))}],["onPlayResult",function(n,t){return e(i("onPlayResult",u.decode(t)))}],["onAlert",function(n,t){return e(i("onAlert",r.decode(t)))}],["onRoundInvalid",function(n,t){return e(i("onRoundInvalid",l.decode(t)))}],["onSettlement",function(n,t){return e(i("onSettlement",s.decode(t)))}],["onError",function(n,t){return e(i("onError",c.decode(t)))}]];d<f.length;d++){var h=f[d],P=h[0],v=h[1];n.on(P,v)}})),e._RF.push({},"fd5b12fX5RNkoMOnv4fJvT7","DawuguiPushHandler",void 0),e._RF.pop()}}}));

System.register("chunks:///_virtual/GameEntry.ts",["cc","./DawuguiModule.ts"],(function(){var t,e;return{setters:[function(e){t=e.cclegacy},function(t){e=t.registerDawuguiModule}],execute:function(){t._RF.push({},"a3b92mhU1dCgrEGiVZdNztk","GameEntry",void 0),e(),t._RF.pop()}}}));

System.register("chunks:///_virtual/StandaloneEntry.ts",["./rollupPluginModLoBabelHelpers.js","cc","./GameHost.ts"],(function(n){var t,e,r,a;return{setters:[function(n){t=n.asyncToGenerator,e=n.regeneratorRuntime},function(n){r=n.cclegacy},function(n){a=n.GameHost}],execute:function(){function o(){return(o=t(e().mark((function n(){return e().wrap((function(n){for(;;)switch(n.prev=n.next){case 0:return n.next=2,a.launch({mode:"standalone",gameId:"dawugui"});case 2:case"end":return n.stop()}}),n)})))).apply(this,arguments)}n("bootstrapStandalone",(function(){return o.apply(this,arguments)})),r._RF.push({},"a100fSxtgdJWIXvV9/1V/Tz","StandaloneEntry",void 0),r._RF.pop()}}}));

(function(r) {
  r('virtual:///prerequisite-imports/dawugui', 'chunks:///_virtual/dawugui'); 
})(function(mid, cid) {
    System.register(mid, [cid], function (_export, _context) {
    return {
        setters: [function(_m) {
            var _exportObj = {};

            for (var _key in _m) {
              if (_key !== "default" && _key !== "__esModule") _exportObj[_key] = _m[_key];
            }
      
            _export(_exportObj);
        }],
        execute: function () { }
    };
    });
});