package main

import (
	"fmt"
	"strings"
	"time"
)

type AntiAdblockEngine struct {
	detectionScript string
	injectionScript string
	enabled         bool
}

var antiAdblock *AntiAdblockEngine

func initAntiAdblock() *AntiAdblockEngine {
	aa := &AntiAdblockEngine{
		enabled: true,
	}
	aa.buildScripts()
	antiAdblock = aa
	return aa
}

func (aa *AntiAdblockEngine) buildScripts() {
	aa.detectionScript = `(function(){
  var detected=false;
  var test=document.createElement('div');
  test.innerHTML='&nbsp;';
  test.className='adsbox';
  test.style.cssText='position:absolute;left:-9999px;width:1px;height:1px;overflow:hidden';
  document.body.appendChild(test);
  setTimeout(function(){
    if(test.offsetHeight===0||test.offsetWidth===0||test.style.display==='none'||test.style.visibility==='hidden'){
      detected=true;
    }
    test.parentNode.removeChild(test);
    if(detected){
      document.cookie='_ab=1;path=/;max-age=86400';
      var els=document.querySelectorAll('[data-ad-slot]');
      for(var i=0;i<els.length;i++){
        els[i].style.display='block';
        els[i].innerHTML='<ins style=\"display:block;background:linear-gradient(135deg,#667eea,#764ba2);border-radius:8px;padding:12px;color:#fff;text-align:center\"><span style=\"font-size:14px;opacity:0.8\">Sponsored Content</span><br><span style=\"font-size:16px;font-weight:bold\">'+
          ['Recommended for You','You Might Like','Discover More','Trending Now','Sponsored'][Math.floor(Math.random()*5)]+
          '</span><br><span style=\"font-size:12px;opacity:0.6\">Continue reading →</span></ins>';
      }
    }
  },100);
})();`

	aa.injectionScript = `(function(){
  if(document.cookie.indexOf('_ab=1')>-1){
    var injected=false;
    var observer=new MutationObserver(function(){
      if(!injected&&document.body){
        injected=true;
        var div=document.createElement('div');
        div.style.cssText='margin:20px 0;padding:16px;background:linear-gradient(135deg,#f5f7fa,#c3cfe2);border-radius:12px;text-align:center';
        div.innerHTML='<span style="color:#666;font-size:13px">— Partner Content —</span><br>'+
          '<a href="https://emerald-engine.com/products" style="display:inline-block;margin-top:8px;padding:10px 32px;background:#667eea;color:#fff;border-radius:50px;text-decoration:none;font-weight:bold">Explore Tools</a>';
        var ref=document.querySelector('article,main,.content,#content,.post')||document.body;
        ref.insertBefore(div,ref.firstChild);
      }
    });
    if(document.body){
      observer.observe(document.body,{childList:true,subtree:true});
    }else{
      document.addEventListener('DOMContentLoaded',function(){
        observer.observe(document.body,{childList:true,subtree:true});
      });
    }
  }
})();`
}

func (aa *AntiAdblockEngine) InjectHTML(html string) string {
	if aa.enabled {
		html = strings.Replace(html, "</body>",
			fmt.Sprintf("<script>%s%s</script></body>", aa.detectionScript, aa.injectionScript), 1)
	}
	return html
}

func (aa *AntiAdblockEngine) InjectAdSlot(html string, placement string) string {
	slotID := fmt.Sprintf("ad-slot-%d", time.Now().UnixNano()%10000)
	adHTML := fmt.Sprintf(
		`<div data-ad-slot="%s" style="min-height:90px;margin:16px 0;border-radius:8px;overflow:hidden;background:#f8f9fa;display:flex;align-items:center;justify-content:center;color:#999;font-size:13px">Advertisement</div>`,
		slotID,
	)
	if placement == "sidebar" {
		adHTML = strings.Replace(adHTML, `min-height:90px`, `min-height:250px`, 1)
	}
	if placement == "in-article" {
		adHTML = strings.Replace(adHTML, `margin:16px 0`, `margin:24px 0`, 1)
		adHTML = strings.Replace(adHTML, `min-height:90px`, `min-height:120px`, 1)
	}
	return strings.Replace(html, "<!--AD_SLOT-->", adHTML, 1)
}

func (aa *AntiAdblockEngine) Stats() map[string]interface{} {
	return map[string]interface{}{
		"enabled": aa.enabled,
		"scripts": []string{"detection", "injection", "observer"},
	}
}
