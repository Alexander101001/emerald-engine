package main

import (
	"fmt"
)

func genTradingIndex() string {
	title := "Crypto Trading Platform - Automated Trading Bots & Signals"
	desc := "Professional crypto trading platform with grid bots, trading signals, portfolio tracker, copy trading, and arbitrage scanner. Connect your Binance account and start trading."
	db := tradingDB
	stats := ""
	if db != nil {
		stats = fmt.Sprintf(`<div class="g" style="max-width:800px;margin:30px auto"><div class="acd" style="text-align:center"><div style="font-size:2.5em;color:#667eea;font-weight:bold">%d</div><p style="color:#666">Total Users</p></div><div class="acd" style="text-align:center"><div style="font-size:2.5em;color:#2ecc71;font-weight:bold">%d</div><p style="color:#666">Active Traders</p></div><div class="acd" style="text-align:center"><div style="font-size:2.5em;color:#f39c12;font-weight:bold">$%.0f</div><p style="color:#666">Monthly Revenue</p></div></div>`,
			db.totalUsers(), db.payingUsers(), db.monthlyRevenue())
	}

	tiers := `<div class="g" style="max-width:1000px;margin:40px auto">
<div class="acd" style="text-align:center;border:2px solid #e0e0e0"><h3>🚀 Starter</h3><div class="pr" style="font-size:2em;margin:15px 0">$19<span style="font-size:0.4em;color:#999">/mo</span></div><ul style="text-align:left;list-style:none;line-height:2.2;color:#555"><li>✅ 1 Grid Bot</li><li>✅ Basic Signals</li><li>✅ Portfolio Tracker</li><li>✅ 5 Price Alerts</li><li>❌ Copy Trading</li><li>❌ API Access</li></ul><a href="#" class="btn btn-p" style="margin-top:15px;display:inline-block">Get Started →</a></div>
<div class="acd" style="text-align:center;border:2px solid #667eea;box-shadow:0 8px 30px rgba(102,126,234,0.2);position:relative"><div class="bg" style="position:absolute;top:-12px;left:50%;transform:translateX(-50%)">🔥 Most Popular</div><h3>💎 Pro</h3><div class="pr" style="font-size:2em;margin:15px 0">$49<span style="font-size:0.4em;color:#999">/mo</span></div><ul style="text-align:left;list-style:none;line-height:2.2;color:#555"><li>✅ 5 Grid Bots</li><li>✅ All Strategies</li><li>✅ Signal Marketplace</li><li>✅ Copy Trading (3 traders)</li><li>✅ 20 Alerts</li><li>✅ Advanced Analytics</li></ul><a href="#" class="btn btn-p" style="margin-top:15px;display:inline-block">Get Started →</a></div>
<div class="acd" style="text-align:center;border:2px solid #e0e0e0"><h3>🏢 Enterprise</h3><div class="pr" style="font-size:2em;margin:15px 0">$199<span style="font-size:0.4em;color:#999">/mo</span></div><ul style="text-align:left;list-style:none;line-height:2.2;color:#555"><li>✅ Unlimited Bots</li><li>✅ All Features</li><li>✅ White-label Signals</li><li>✅ API Access</li><li>✅ Priority Support</li><li>✅ Dedicated Server</li></ul><a href="#" class="btn btn-p" style="margin-top:15px;display:inline-block">Get Started →</a></div>
</div>`

	return fmt.Sprintf(`%s%s<div class="container">
<header style="text-align:center;padding:60px 0 40px;background:linear-gradient(135deg,#1a1a2e,#16213e);border-radius:0 0 30px 30px;color:white;margin:-20px -20px 0"><h1 style="font-size:3em;margin-bottom:15px">🤖 Emerald Trading</h1><p style="font-size:1.2em;opacity:0.9;max-width:700px;margin:0 auto">Automated crypto trading platform. Connect Binance. Trade 24/7.</p><div style="margin-top:25px"><a href="/trading/register" class="btn btn-p" style="background:#e94560;padding:16px 50px;font-size:1.2em">Start Free →</a><a href="/trading/login" style="color:white;margin-left:20px;opacity:0.8">Sign In</a></div></header>
%s
<h2 style="text-align:center;margin:50px 0 20px;color:#333">Choose Your Plan</h2>
%s
<h2 style="text-align:center;margin:50px 0 20px;color:#333">Live Market Data</h2>
<div class="ac">%s</div>
<div id="market-table" style="overflow-x:auto"><table style="width:100%%;border-collapse:collapse;margin:20px 0"><thead><tr style="background:#f8f9fa"><th style="padding:12px;text-align:left">Symbol</th><th style="padding:12px;text-align:right">Price</th><th style="padding:12px;text-align:right">24h Change</th><th style="padding:12px;text-align:right">24h Volume</th></tr></thead><tbody id="market-data"><tr><td colspan="4" style="text-align:center;padding:30px;color:#999">Loading market data...</td></tr></tbody></table></div>
<script>
fetch('/api/market').then(r=>r.json()).then(d=>{var h='';d.tickers.forEach(function(t){var c=parseFloat(t.priceChange)>0?'#2ecc71':'#e74c3c';h+='<tr><td style="padding:10px;border-bottom:1px solid #eee"><strong>'+t.symbol+'</strong></td><td style="padding:10px;border-bottom:1px solid #eee;text-align:right">$'+parseFloat(t.lastPrice).toFixed(t.lastPrice<1?6:2)+'</td><td style="padding:10px;border-bottom:1px solid #eee;text-align:right;color:'+c+'">'+parseFloat(t.priceChange).toFixed(2)+' ('+(parseFloat(t.priceChange)/parseFloat(t.lastPrice)*100).toFixed(2)+'%)</td><td style="padding:10px;border-bottom:1px solid #eee;text-align:right">$'+(parseFloat(t.quoteVolume)/1e6).toFixed(2)+'M</td></tr>'});document.getElementById('market-data').innerHTML=h});
</script>
<div class="ac">%s</div>
%s
<div class="ac">%s</div>
<div style="text-align:center;margin:40px 0"><a href="/trading/register" class="btn btn-p" style="font-size:1.3em;padding:18px 60px;background:#667eea">Create Free Account →</a><p style="color:#999;margin-top:12px">No credit card required. Connect your Binance API key to start.</p></div>%s`,
		commonHead(title, desc), navBar(Niche{Name: "Trading", Keyword: "crypto-trading", Emoji: "🤖"}),
		stats,
		tiers,
		adBanner(),
		dynamicEmailForm(),
		adInArticle(),
		cryptoDonate(),
		foot())
}

func genTradingRegister() string {
	title := "Create Account - Emerald Trading Platform"
	desc := "Register for free on Emerald Trading. Connect your Binance API key and start automated trading."
	return fmt.Sprintf(`%s%s<div class="container" style="max-width:500px">
<header style="text-align:center;padding:40px 0 20px"><h1 style="font-size:2.2em;color:#333">Create Account</h1><p style="color:#666">Free tier includes portfolio tracker</p></header>
<div class="rb" style="max-width:450px;margin:0 auto">
<form id="register-form"><div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Email</label><input type="email" id="reg-email" placeholder="your@email.com" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1em"></div>
<div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Password</label><input type="password" id="reg-password" placeholder="Min 8 characters" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1em"></div>
<button type="submit" class="btn btn-p" style="width:100%%;font-size:1.1em;margin-top:10px">Create Free Account →</button></form>
<p style="text-align:center;margin-top:15px;color:#666">Already have an account? <a href="/trading/login">Sign In</a></p>
<div id="reg-result" style="margin-top:20px;padding:15px;border-radius:8px;text-align:center;display:none"></div></div>
<script>
document.getElementById('register-form').onsubmit=function(e){e.preventDefault();var email=document.getElementById('reg-email').value,pwd=document.getElementById('reg-password').value;fetch('/api/register',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({email:email,password:pwd})}).then(function(r){return r.json()}).then(function(d){if(d.error){document.getElementById('reg-result').style.display='block';document.getElementById('reg-result').style.background='#fdf2f2';document.getElementById('reg-result').innerHTML='❌ '+d.error}else{document.getElementById('reg-result').style.display='block';document.getElementById('reg-result').style.background='#e8f8f5';document.getElementById('reg-result').innerHTML='✅ Account created! <a href="/trading/login">Sign in →</a>';localStorage.setItem('trading_token',d.token);localStorage.setItem('trading_email',d.email)}})};
</script>
</div>%s`,
		commonHead(title, desc), navBar(Niche{Name: "Trading", Keyword: "crypto-trading", Emoji: "🤖"}),
		foot())
}

func genTradingLogin() string {
	title := "Sign In - Emerald Trading Platform"
	desc := "Sign in to your Emerald Trading account. Access your portfolio, bots, and trading signals."
	return fmt.Sprintf(`%s%s<div class="container" style="max-width:500px">
<header style="text-align:center;padding:40px 0 20px"><h1 style="font-size:2.2em;color:#333">Sign In</h1><p style="color:#666">Welcome back to Emerald Trading</p></header>
<div class="rb" style="max-width:450px;margin:0 auto">
<form id="login-form"><div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Email</label><input type="email" id="login-email" placeholder="your@email.com" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1em"></div>
<div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Password</label><input type="password" id="login-password" placeholder="Enter password" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1em"></div>
<button type="submit" class="btn btn-p" style="width:100%%;font-size:1.1em;margin-top:10px">Sign In →</button></form>
<p style="text-align:center;margin-top:15px;color:#666">No account? <a href="/trading/register">Create one</a></p>
<div id="login-result" style="margin-top:20px;padding:15px;border-radius:8px;text-align:center;display:none"></div></div>
<script>
document.getElementById('login-form').onsubmit=function(e){e.preventDefault();var email=document.getElementById('login-email').value,pwd=document.getElementById('login-password').value;fetch('/api/login',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({email:email,password:pwd})}).then(function(r){return r.json()}).then(function(d){if(d.error){document.getElementById('login-result').style.display='block';document.getElementById('login-result').style.background='#fdf2f2';document.getElementById('login-result').innerHTML='❌ '+d.error}else{document.getElementById('login-result').style.display='block';document.getElementById('login-result').style.background='#e8f8f5';document.getElementById('login-result').innerHTML='✅ Signed in! <a href="/trading/dashboard">Go to Dashboard →</a>';localStorage.setItem('trading_token',d.token);localStorage.setItem('trading_email',d.email)}})};
</script>
</div>%s`,
		commonHead(title, desc), navBar(Niche{Name: "Trading", Keyword: "crypto-trading", Emoji: "🤖"}),
		foot())
}

func genTradingDashboard() string {
	title := "Trading Dashboard - Portfolio, Bots & Signals"
	desc := "Your Emerald Trading dashboard. View portfolio, manage bots, check signals, and track performance."
	return fmt.Sprintf(`%s%s<div class="container">
<header style="text-align:center;padding:30px 0 20px"><h1 style="font-size:2em;color:#333">📊 Trading Dashboard</h1><p style="color:#666">Your portfolio, bots, and performance at a glance</p></header>
<div class="g" style="grid-template-columns:repeat(auto-fit,minmax(200px,1fr))">
<div class="acd" style="text-align:center"><div style="font-size:1.5em;color:#667eea;font-weight:bold" id="dash-total">$0.00</div><p style="color:#999;font-size:0.85em">Total Portfolio</p></div>
<div class="acd" style="text-align:center"><div style="font-size:1.5em;color:#2ecc71;font-weight:bold" id="dash-pnl24h">$0.00</div><p style="color:#999;font-size:0.85em">24h P&L</p></div>
<div class="acd" style="text-align:center"><div style="font-size:1.5em;color:#f39c12;font-weight:bold" id="dash-bots">0</div><p style="color:#999;font-size:0.85em">Active Bots</p></div>
<div class="acd" style="text-align:center"><div style="font-size:1.5em;color:#e94560;font-weight:bold" id="dash-signals">0</div><p style="color:#999;font-size:0.85em">New Signals</p></div>
</div>
<div class="ac">%s</div>
<div class="tc"><a href="/trading/grid" class="btn btn-p">⚡ Create Grid Bot</a><a href="/trading/signals" class="btn btn-s">📡 View Signals</a><a href="/trading/copytrade" class="btn btn-w">👥 Copy Trading</a></div>
<h2 style="margin:30px 0 15px;color:#333">Your Bots</h2>
<div id="bots-list"><p style="color:#999;text-align:center;padding:20px">No active bots. <a href="/trading/grid">Create your first grid bot →</a></p></div>
<div class="ac">%s</div>
<h2 style="margin:30px 0 15px;color:#333">Recent Signals</h2>
<div id="signals-list"><p style="color:#999;text-align:center;padding:20px">Loading signals...</p></div>
<div class="ac">%s</div>
<script>
var email=localStorage.getItem('trading_email');
if(email){
fetch('/api/portfolio?email='+email).then(r=>r.json()).then(function(d){if(d.total_usdt){document.getElementById('dash-total').innerHTML='$'+d.total_usdt.toFixed(2);document.getElementById('dash-pnl24h').innerHTML='$'+(d.pnl_24h||0).toFixed(2)}}).catch(function(){});
fetch('/api/bots?email='+email).then(r=>r.json()).then(function(d){if(d.length){document.getElementById('dash-bots').innerHTML=d.length;var h='';d.forEach(function(b){h+='<div class="rb" style="padding:20px;margin:10px 0"><div style="display:flex;justify-content:space-between"><strong>'+b.type.toUpperCase()+'</strong><span style="color:'+(b.status=='running'?'#2ecc71':'#e74c3c')+'">'+b.status+'</span></div><p>'+b.symbol+' | P&L: $'+b.pnl.toFixed(2)+' | Trades: '+b.trade_count+'</p></div>'});document.getElementById('bots-list').innerHTML=h}}).catch(function(){});
}
fetch('/api/signals').then(r=>r.json()).then(function(d){if(d.length){document.getElementById('dash-signals').innerHTML=d.length;var h='';d.slice(0,10).forEach(function(s){h+='<div class="rb" style="padding:15px;margin:8px 0;display:flex;justify-content:space-between;align-items:center"><div><strong style="color:'+(s.side=='BUY'?'#2ecc71':'#e74c3c')+'">'+s.side+'</strong> '+s.symbol+'<br><small style="color:#999">'+s.strategy+' | $'+s.price.toFixed(s.price<1?6:2)+'</small></div><span class="bg" style="background:'+(s.strength>70?'#2ecc71':'#f39c12')+'">'+s.strength.toFixed(0)+'%</span></div>'});document.getElementById('signals-list').innerHTML=h}}).catch(function(){document.getElementById('signals-list').innerHTML='<p style="color:#999;text-align:center;padding:20px">Connect Binance API to see signals</p>'});
</script>
%s`,
		commonHead(title, desc), navBar(Niche{Name: "Trading", Keyword: "crypto-trading", Emoji: "🤖"}),
		adBanner(),
		adInArticle(),
		adSidebar(),
		dynamicEmailForm(),
		foot())
}

func genTradingGrid() string {
	title := "Grid Trading Bot - Automated Crypto Trading"
	desc := "Create automated grid trading bots on Binance. Set price range, grid count, and let the bot trade 24/7."
	return fmt.Sprintf(`%s%s<div class="container" style="max-width:700px">
<header style="text-align:center;padding:40px 0 20px"><h1 style="font-size:2.2em;color:#333">⚡ Grid Trading Bot</h1><p style="color:#666">Set your parameters and let the bot trade 24/7</p></header>
<div class="ac">%s</div>
<div class="rb"><form id="grid-form"><div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Trading Pair</label>
<select id="grid-symbol" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1em"><option value="BTCUSDT">BTC/USDT</option><option value="ETHUSDT">ETH/USDT</option><option value="BNBUSDT">BNB/USDT</option><option value="SOLUSDT">SOL/USDT</option><option value="XRPUSDT">XRP/USDT</option><option value="ADAUSDT">ADA/USDT</option><option value="DOGEUSDT">DOGE/USDT</option><option value="AVAXUSDT">AVAX/USDT</option></select></div>
<div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Lower Price ($)</label><input type="number" id="grid-low" value="50000" step="0.01" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1em"></div>
<div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Upper Price ($)</label><input type="number" id="grid-high" value="80000" step="0.01" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1em"></div>
<div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Number of Grids</label><input type="number" id="grid-count" value="10" min="2" max="50" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1em"></div>
<div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Total Investment (USDT)</label><input type="number" id="grid-invest" value="1000" min="10" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1em"></div>
<button type="submit" class="btn btn-p" style="width:100%%;font-size:1.2em;padding:16px">Start Grid Bot →</button></form>
<div id="grid-result" style="margin-top:20px;padding:20px;border-radius:10px;text-align:center;display:none"></div></div>
<div class="ac">%s</div>
<p style="line-height:1.8;color:#444">Grid trading automatically buys low and sells high within your set price range. As price fluctuates, the bot places limit orders at each grid level. Each time price crosses a grid line, you profit from the difference.</p>
<div class="ac">%s</div>%s<div class="ac">%s</div><div style="margin:40px 0;text-align:center"><a href="/trading/dashboard" class="btn btn-p">← Back to Dashboard</a></div>
<script>
document.getElementById('grid-form').onsubmit=function(e){e.preventDefault();
var email=localStorage.getItem('trading_email');if(!email){document.getElementById('grid-result').style.display='block';document.getElementById('grid-result').style.background='#fdf2f2';document.getElementById('grid-result').innerHTML='❌ Please <a href="/trading/login">sign in</a> first';return}
fetch('/api/grid/create',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({email:email,symbol:document.getElementById('grid-symbol').value,lower_price:parseFloat(document.getElementById('grid-low').value),upper_price:parseFloat(document.getElementById('grid-high').value),grid_count:parseInt(document.getElementById('grid-count').value),total_usdt:parseFloat(document.getElementById('grid-invest').value)})}).then(function(r){return r.json()}).then(function(d){if(d.error){document.getElementById('grid-result').style.display='block';document.getElementById('grid-result').style.background='#fdf2f2';document.getElementById('grid-result').innerHTML='❌ '+d.error}else{document.getElementById('grid-result').style.display='block';document.getElementById('grid-result').style.background='#e8f8f5';document.getElementById('grid-result').innerHTML='✅ Grid bot started! <a href="/trading/dashboard">View in Dashboard →</a>'}})};
</script>
%s`,
		commonHead(title, desc), navBar(Niche{Name: "Trading", Keyword: "crypto-trading", Emoji: "🤖"}),
		adBanner(),
		adInArticle(),
		adSidebar(),
		dynamicEmailForm(),
		adMatched(),
		foot())
}

func genTradingSignals() string {
	title := "Trading Signals - Real-time Buy & Sell Signals"
	desc := "Real-time crypto trading signals generated by AI. RSI, MACD, EMA crossover strategies across 15+ trading pairs."
	return fmt.Sprintf(`%s%s<div class="container">
<header style="text-align:center;padding:40px 0 20px"><h1 style="font-size:2.2em;color:#333">📡 Trading Signals</h1><p style="color:#666">AI-generated buy/sell signals across 15+ trading pairs</p></header>
<div class="ac">%s</div>
<div class="tc"><span class="bg" style="background:#2ecc71">BUY</span><span class="bg" style="background:#e74c3c">SELL</span><span class="bg" style="background:#667eea">Signal Strength</span></div>
<div id="signals-container"><p style="text-align:center;padding:30px;color:#999">Loading signals...</p></div>
<div class="ac">%s</div>%s<div class="ac">%s</div><div style="margin:40px 0;text-align:center"><a href="/trading/dashboard" class="btn btn-p">← Back to Dashboard</a></div>
<script>
function loadSignals(){fetch('/api/signals').then(function(r){return r.json()}).then(function(d){if(!d||!d.length){document.getElementById('signals-container').innerHTML='<p style="text-align:center;padding:30px;color:#999">No active signals. Check back soon.</p>';return}
var h='<table style="width:100%%;border-collapse:collapse"><thead><tr style="background:#f8f9fa"><th style="padding:12px;text-align:left">Signal</th><th style="padding:12px;text-align:left">Strategy</th><th style="padding:12px;text-align:right">Price</th><th style="padding:12px;text-align:center">Strength</th><th style="padding:12px;text-align:right">Time</th></tr></thead><tbody>';
d.forEach(function(s){var c=s.side=='BUY'?'#2ecc71':'#e74c3c';var st=new Date(s.timestamp);h+='<tr><td style="padding:10px;border-bottom:1px solid #eee"><strong style="color:'+c+'">'+s.side+'</strong> '+s.symbol+'</td><td style="padding:10px;border-bottom:1px solid #eee">'+s.strategy+'</td><td style="padding:10px;border-bottom:1px solid #eee;text-align:right">$'+s.price.toFixed(s.price<1?6:2)+'</td><td style="padding:10px;border-bottom:1px solid #eee;text-align:center"><span class="bg" style="background:'+(s.strength>70?'#2ecc71':'#f39c12')+'">'+s.strength.toFixed(0)+'%</span></td><td style="padding:10px;border-bottom:1px solid #eee;text-align:right;font-size:0.85em;color:#999">'+st.toLocaleTimeString()+'</td></tr>'});h+='</tbody></table><p style="text-align:center;color:#999;margin-top:10px;font-size:0.85em">Auto-refreshes every 5 minutes</p>';document.getElementById('signals-container').innerHTML=h})}
loadSignals();setInterval(loadSignals,300000);
</script>
%s`,
		commonHead(title, desc), navBar(Niche{Name: "Trading", Keyword: "crypto-trading", Emoji: "🤖"}),
		adBanner(),
		adInArticle(),
		dynamicEmailForm(),
		adSidebar(),
		foot())
}

func genTradingCopyTrade() string {
	title := "Copy Trading - Follow Top Traders"
	desc := "Copy trades from the best performing traders on Emerald Trading. Automatically mirror their positions."
	return fmt.Sprintf(`%s%s<div class="container">
<header style="text-align:center;padding:40px 0 20px"><h1 style="font-size:2.2em;color:#333">👥 Copy Trading</h1><p style="color:#666">Follow top traders and automatically copy their trades</p></header>
<div class="ac">%s</div>
<h2 style="margin:30px 0 15px;color:#333">🏆 Top Traders</h2>
<div id="leaderboard"><p style="text-align:center;padding:20px;color:#999">Loading leaderboard...</p></div>
<div class="ac">%s</div>
<div class="rb" style="max-width:500px;margin:30px auto"><h3 style="text-align:center">Follow a Trader</h3><form id="copy-form"><div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Trader User ID</label><input type="text" id="copy-leader" placeholder="Enter trader's user ID" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1em"></div>
<div style="margin:15px 0"><label style="display:block;margin-bottom:5px;font-weight:bold">Allocation (USDT)</label><input type="number" id="copy-amount" value="500" min="50" style="width:100%%;padding:12px;border:2px solid #ddd;border-radius:8px;font-size:1em"></div>
<button type="submit" class="btn btn-p" style="width:100%%;font-size:1.1em">Start Copy Trading →</button></form>
<div id="copy-result" style="margin-top:20px;display:none"></div></div>
<div class="ac">%s</div>%s<div style="margin:40px 0;text-align:center"><a href="/trading/dashboard" class="btn btn-p">← Back to Dashboard</a></div>
<script>
fetch('/api/copytrade/leaderboard').then(function(r){return r.json()}).then(function(d){if(!d||!d.length){document.getElementById('leaderboard').innerHTML='<p style="text-align:center;padding:30px;color:#999">No traders on the leaderboard yet. Be the first!</p>';return}
var h='<table style="width:100%%;border-collapse:collapse"><thead><tr style="background:#f8f9fa"><th style="padding:12px;text-align:left">Trader</th><th style="padding:12px;text-align:right">P&L</th><th style="padding:12px;text-align:right">Win Rate</th><th style="padding:12px;text-align:right">Trades</th><th style="padding:12px;text-align:right">Followers</th></tr></thead><tbody>';
d.forEach(function(t){h+='<tr><td style="padding:10px;border-bottom:1px solid #eee">'+t.user_id.substring(0,8)+'...</td><td style="padding:10px;border-bottom:1px solid #eee;text-align:right;color:'+(t.pnl>=0?'#2ecc71':'#e74c3c')+'">$'+t.pnl.toFixed(2)+'</td><td style="padding:10px;border-bottom:1px solid #eee;text-align:right">'+(t.win_rate*100).toFixed(1)+'%</td><td style="padding:10px;border-bottom:1px solid #eee;text-align:right">'+t.trade_count+'</td><td style="padding:10px;border-bottom:1px solid #eee;text-align:right">'+t.followers+'</td></tr>'});h+='</tbody></table>';document.getElementById('leaderboard').innerHTML=h});
document.getElementById('copy-form').onsubmit=function(e){e.preventDefault();var email=localStorage.getItem('trading_email');if(!email){document.getElementById('copy-result').style.display='block';document.getElementById('copy-result').innerHTML='❌ Please sign in first';return}
fetch('/api/copytrade/follow',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({follower:email,leader:document.getElementById('copy-leader').value,amount:parseFloat(document.getElementById('copy-amount').value)})}).then(function(r){return r.json()}).then(function(d){document.getElementById('copy-result').style.display='block';if(d.error){document.getElementById('copy-result').innerHTML='❌ '+d.error}else{document.getElementById('copy-result').innerHTML='✅ Now following trader!'}})};
</script>
%s`,
		commonHead(title, desc), navBar(Niche{Name: "Trading", Keyword: "crypto-trading", Emoji: "🤖"}),
		adBanner(),
		adInArticle(),
		adSidebar(),
		dynamicEmailForm(),
		foot())
}

func genTradingPages() map[string]func() string {
	return map[string]func() string{
		"public/trading/index.html":    genTradingIndex,
		"public/trading/register.html": genTradingRegister,
		"public/trading/login.html":    genTradingLogin,
		"public/trading/dashboard.html": genTradingDashboard,
		"public/trading/grid.html":     genTradingGrid,
		"public/trading/signals.html":  genTradingSignals,
		"public/trading/copytrade.html": genTradingCopyTrade,
	}
}
