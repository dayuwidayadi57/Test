package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"math/big"
	"net/http"
	"os"
	"strings"
	"strconv"
	"sync"
	"time"
)

const (
	RPC         = "https://octra.network"
	MicroOCT    = 1000000
	B58         = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	DB_FILE     = "gamefi_db.json"
	CONFIG_FILE = "admin_config.json"
	ADMIN_KEY   = "rahasia"
)

type AdminConfig struct {
	HouseWalletAddr string         `json:"house_wallet_address"`
	HousePrivateKey string         `json:"house_private_key"`
	WithdrawFee     int64          `json:"withdraw_fee_microoct"`
	MinWithdraw     int64          `json:"min_withdraw"`
	IsMaintenance   bool           `json:"is_maintenance"`
	RTPWeights      map[string]int `json:"rtp_weights"`
}

type WDRequest struct {
	ID        string `json:"id"`
	Address   string `json:"address"`
	Amount    int64  `json:"amount"`
	Status    string `json:"status"`
	TxHash    string `json:"tx_hash"`
	Timestamp int64  `json:"timestamp"`
}

type PlayerData struct {
	Address          string `json:"address"`
	VirtualBalance   int64  `json:"virtual_balance"`
	TotalDeposited   int64  `json:"total_deposited"`
	TotalWithdrawn   int64  `json:"total_withdrawn"`
	TotalSpins       int    `json:"total_spins"`
	TotalWins        int    `json:"total_wins"`
	LastCheckIn      int64  `json:"last_check_in"`
	FreeSpinsLeft    int    `json:"free_spins_left"`
	CheckInStreak    int    `json:"check_in_streak"`
}

type GameDatabase struct {
	Players     map[string]*PlayerData `json:"players"`
	Withdraws   []*WDRequest           `json:"withdraws"`
	HouseProfit int64                  `json:"house_profit"`
	TotalSpins  int                    `json:"total_spins"`
}

type WinPos struct {
	Row int `json:"r"`
	Col int `json:"c"`
}

var (
	DB       *GameDatabase
	AdminCfg *AdminConfig
	mu       sync.RWMutex
	PAYOUTS  = map[string]int{"🍒": 2, "🍋": 3, "🍊": 5, "🍇": 10, "💎": 25, "⭐": 50, "🎰": 150}
	funcMap  = template.FuncMap{
		"div": func(a, b int64) float64 {
			if b == 0 { return 0 }
			return float64(a) / float64(b)
		},
	}
)

func Base58Encode(input []byte) string {
	x := new(big.Int).SetBytes(input)
	base := big.NewInt(58)
	zero := big.NewInt(0)
	var res []byte
	for x.Cmp(zero) > 0 {
		mod := new(big.Int)
		x.DivMod(x, base, mod)
		res = append(res, B58[mod.Int64()])
	}
	for i, j := 0, len(res)-1; i < j; i, j = i+1, j-1 {
		res[i], res[j] = res[j], res[i]
	}
	return string(res)
}

func GetAddr(privBase64 string) (string, string, ed25519.PrivateKey) {
	seed, _ := base64.StdEncoding.DecodeString(privBase64)
	if len(seed) != 32 { return "", "", nil }
	priv := ed25519.NewKeyFromSeed(seed)
	pub := priv.Public().(ed25519.PublicKey)
	hash := sha256.Sum256(pub)
	return "oct" + Base58Encode(hash[:]), base64.StdEncoding.EncodeToString(pub), priv
}

func getAccountData(addr string) (float64, int) {
	resp, err := http.Get(RPC + "/balance/" + addr)
	if err != nil { return 0, 0 }
	defer resp.Body.Close()
	var data struct {
		Balance interface{} `json:"balance"`
		Nonce   int         `json:"nonce"`
	}
	json.NewDecoder(resp.Body).Decode(&data)
	var bal float64
	switch v := data.Balance.(type) {
	case float64: bal = v
	case string: fmt.Sscanf(v, "%f", &bal)
	}
	r2, err := http.Get(RPC + "/staging")
	if err == nil {
		defer r2.Body.Close()
		var stg struct { Txs []struct{ From string; Nonce int } `json:"staged_transactions"` }
		json.NewDecoder(r2.Body).Decode(&stg)
		for _, tx := range stg.Txs {
			if strings.EqualFold(tx.From, addr) && tx.Nonce > data.Nonce {
				data.Nonce = tx.Nonce
			}
		}
	}
	return bal, data.Nonce
}

func sendTX(privB64, to string, amt int64) (string, error) {
	from, pub, priv := GetAddr(privB64)
	if priv == nil { 
		return "", fmt.Errorf("invalid private key format")
	}
	
	_, nonce := getAccountData(from)
	ts := float64(time.Now().UnixNano()) / 1e9
	
	msg := fmt.Sprintf(`{"from":"%s","to_":"%s","amount":"%d","nonce":%d,"ou":"1000","timestamp":%.7f}`,
		from, to, amt, nonce+1, ts)
	
	sig := base64.StdEncoding.EncodeToString(ed25519.Sign(priv, []byte(msg)))
	
	payload := map[string]interface{}{
		"from": from, "to_": to, "amount": fmt.Sprintf("%d", amt),
		"nonce": nonce + 1, "ou": "1000", "timestamp": ts,
		"signature": sig, "public_key": pub,
	}
	
	body, _ := json.Marshal(payload)
	resp, err := http.Post(RPC+"/send-tx", "application/json", bytes.NewBuffer(body))
	if err != nil { 
		return "", fmt.Errorf("http error: %v", err)
	}
	defer resp.Body.Close()

	var resData struct {
		Hash   string `json:"tx_hash"`
		Status string `json:"status"`
		Error  string `json:"error"`
	}
	json.NewDecoder(resp.Body).Decode(&resData)
	
	if resData.Error != "" {
		return "", fmt.Errorf("rpc error: %s", resData.Error)
	}
	
	if resData.Hash != "" || resData.Status == "accepted" {
		return resData.Hash, nil
	}
	
	return "", fmt.Errorf("unknown error: no hash returned")
}

var tmplPlayer = template.Must(template.New("player").Parse(`
<!DOCTYPE html>
<html>
<head>
    <title>OCTRA CASINO V7.7 - GATES OF OCTRA</title>
    <style>
        body { background: radial-gradient(circle, #0a1e2e 0%, #0b0d1d 100%); color: #fff; font-family: 'Verdana', sans-serif; text-align: center; margin: 0; overflow-x: hidden; }
        .navbar { background: rgba(21, 21, 26, 0.9); padding: 15px 50px; border-bottom: 2px solid #3498db; display: flex; justify-content: space-between; align-items: center; box-shadow: 0 0 20px rgba(52, 152, 219, 0.3); }
        .container { max-width: 900px; margin: 20px auto; background: rgba(21, 21, 26, 0.8); padding: 25px; border-radius: 20px; border: 2px solid #2c5f8d; backdrop-filter: blur(10px); }
        
        .msg-banner { background: rgba(0,0,0,0.6); margin-bottom: 15px; padding: 15px; border-radius: 15px; border: 1px solid #3498db; min-height: 50px; display: flex; align-items: center; justify-content: center; overflow: hidden; position: relative; }
        #status-msg { font-size: 1.4rem; font-weight: bold; color: #3498db; text-shadow: 0 0 10px #3498db; letter-spacing: 1px; transition: 0.3s; }

        .grid { display: grid; grid-template-columns: repeat(5, 1fr); gap: 10px; background: linear-gradient(180deg, #1a3d5d 0%, #0a1524 100%); padding: 20px; border-radius: 15px; margin-bottom: 20px; border: 4px solid #3498db; box-shadow: 0 0 30px rgba(52, 152, 219, 0.4); overflow: hidden; }
        .cell-container { background: rgba(0, 0, 0, 0.5); border-radius: 12px; height: 100px; overflow: hidden; border: 1px solid #2c5f8d; position: relative; transition: 0.2s; }
        
        .win-glow { 
            border: 3px solid #3498db !important; 
            box-shadow: 0 0 20px #3498db, inset 0 0 10px #3498db !important;
            animation: pulse-glow 0.5s infinite alternate;
            z-index: 5;
        }
        @keyframes pulse-glow {
            from { opacity: 0.8; transform: scale(1); }
            to { opacity: 1; transform: scale(1.05); }
        }

        .reel { display: flex; flex-direction: column; align-items: center; width: 100%; }
        .symbol { font-size: 3rem; height: 100px; display: flex; align-items: center; justify-content: center; }

        .spinning { animation: blurSpin 0.15s linear infinite; }
        @keyframes blurSpin { 0% { transform: translateY(0); filter: blur(4px); } 100% { transform: translateY(-200px); filter: blur(8px); } }

        .btn { background: linear-gradient(180deg, #3498db 0%, #2980b9 100%); color: #fff; border: none; padding: 15px 35px; font-weight: 900; border-radius: 50px; cursor: pointer; transition: 0.2s; text-transform: uppercase; box-shadow: 0 5px 0 #1c5985; }
        .btn:active { transform: translateY(3px); box-shadow: 0 2px 0 #1c5985; }
        .btn-auto { background: linear-gradient(180deg, #2196F3 0%, #1976D2 100%); color: #fff; font-size: 0.8rem; padding: 8px 15px; box-shadow: 0 3px 0 #0d47a1; border:none; border-radius:5px; cursor:pointer; }
        input { padding: 12px; border-radius: 10px; border: 2px solid #3498db; background: #000; color: #fff; text-align: center; font-weight: bold; width: 100px; }

        .wallet-box { margin-top: 25px; padding: 20px; background: rgba(0,0,0,0.5); border-radius: 15px; border: 1px solid #2c5f8d; display: flex; gap: 10px; justify-content: center; }
        .btn-wallet { padding: 12px 20px; font-size: 0.8rem; color: #fff; border-radius: 10px; border: none; cursor: pointer; font-weight: bold; }
        .btn-dep { background: #27ae60; box-shadow: 0 4px 0 #1e8449; }
        .btn-wd { background: #c0392b; box-shadow: 0 4px 0 #922b21; }

        .paytable { margin-top: 30px; padding: 20px; background: rgba(20, 30, 40, 0.9); border-radius: 15px; border: 1px solid #3498db; }
        .symbol-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 15px; }
        .symbol-item { background: rgba(52, 152, 219, 0.1); padding: 10px; border-radius: 10px; border: 1px solid #2c5f8d; }
        .symbol-item span { display: block; font-size: 2rem; }
        .val { color: #3498db; font-weight: bold; font-size: 0.8rem; }

        #win-overlay { display: none; position: fixed; top: 40%; left: 50%; transform: translate(-50%, -50%); z-index: 1000; pointer-events: none; width: 100%; text-align: center; }
        .win-text { font-size: 6rem; font-weight: 900; text-transform: uppercase; filter: drop-shadow(0 0 50px #000); animation: winAnim 0.7s cubic-bezier(0.175, 0.885, 0.32, 1.275); }
        .win-emoji { font-size: 4rem; display: block; margin-bottom: 10px; }
        
        @keyframes winAnim { 0% { transform: scale(0) rotate(-20deg); opacity: 0; } 100% { transform: scale(1) rotate(0deg); opacity: 1; } }
        .big-win { color: #3498db; text-shadow: 0 0 20px #3498db, 0 0 40px #2196F3; }
        .mega-win { color: #2196F3; text-shadow: 0 0 20px #2196F3, 0 0 40px #3498db; }
        .super-win { color: #1e88e5; text-shadow: 0 0 30px #1565c0; animation: shake 0.3s infinite; }
        @keyframes shake { 0% { transform: translate(3px, 3px); } 50% { transform: translate(-3px, -3px); } 100% { transform: translate(3px, 3px); } }
        
        /* Tambahan untuk daily check-in */
        .checkin-btn {
            background: linear-gradient(180deg, #2ecc71 0%, #27ae60 100%) !important;
            animation: pulse 2s infinite;
        }
        @keyframes pulse {
            0% { transform: scale(1); }
            50% { transform: scale(1.03); }
            100% { transform: scale(1); }
        }
        .claimed-box {
            background: rgba(0,0,0,0.5); 
            padding: 10px; 
            border-radius: 10px; 
            border: 1px solid #555;
        }
    </style>
</head>
<body>
    {{if .Address}}
    <div class="navbar">
        <div style="font-weight:900; color:#3498db; font-size:1.8rem;">⚡ OCTRA GATES</div>
        <div><span style="background:#222; padding:8px 15px; border-radius:20px; border:1px solid #2c5f8d;"><small style="color:#3498db">●</small> {{.Address}}</span> | <a href="/logout" style="color:#e74c3c; text-decoration:none; margin-left:15px; font-weight:bold;">EXIT</a></div>
    </div>
    
    <div id="win-overlay">
        <div id="win-emoji-box" class="win-emoji"></div>
        <div id="win-text-box" class="win-text"></div>
    </div>

    <div class="container">
        <div class="msg-banner">
            <div id="status-msg">GATES OF OCTRA IS READY!</div>
        </div>

        <div style="display:flex; justify-content:space-between; align-items:center; background:rgba(0,0,0,0.4); padding:15px; border-radius:50px; border:1px solid #3498db; margin-bottom:20px;">
            <div style="padding-left:20px;">
                💰 <span style="color:#888; font-size:0.8rem;">CREDITS</span><br>
                <span style="color:#3498db; font-size:1.4rem; font-weight:bold;" id="v-bal">{{printf "%.4f" .BalanceOCT}}</span><small> OCT</small>
            </div>
            <div>
                🎁 <span style="color:#888; font-size:0.8rem;">FREE SPINS</span><br>
                <span style="color:#2ecc71; font-size:1.4rem; font-weight:bold;" id="free-spins">{{.FreeSpinsLeft}}</span><small> left</small>
            </div>
            <div>
                🔥 <span style="color:#888; font-size:0.8rem;">STREAK</span><br>
                <span style="color:#f1c40f; font-size:1.4rem; font-weight:bold;">{{.CheckInStreak}}</span><small> days</small>
            </div>
            <div style="padding-right:20px;">
                🔄 <span style="color:#888; font-size:0.8rem;">AUTO</span><br>
                <span id="v-auto-count" style="font-size:1.4rem; font-weight:bold; color:#2196F3;">0 / 0</span>
            </div>
        </div>

        <div class="grid" id="game-grid">
            {{range $r, $row := .Grid}}{{range $c, $sym := $row}}
            <div class="cell-container" id="cell-{{$r}}-{{$c}}">
                <div class="reel"><div class="symbol">{{$sym}}</div></div>
            </div>
            {{end}}{{end}}
        </div>

        <div style="background:rgba(0,0,0,0.5); padding:20px; border-radius:20px; border:1px solid #2c5f8d;">
            <label style="color:#3498db; font-weight:bold;">BET:</label>
            <input type="number" step="0.1" id="bet-amt" value="0.1" min="0.1">
            <button type="button" class="btn" onclick="doSpin()" style="margin: 0 20px;">SPIN</button>
            <div style="margin-top:20px;">
                <button type="button" class="btn btn-auto" onclick="startAuto(10)">10x</button>
                <button type="button" class="btn btn-auto" onclick="startAuto(25)">25x</button>
                <button type="button" class="btn btn-auto" onclick="startAuto(50)">50x</button>
                <button type="button" class="btn btn-auto" onclick="startAuto(100)">100x</button>
                <button type="button" class="btn btn-auto" style="background:#e74c3c;" onclick="stopAuto()">STOP</button>
            </div>
        </div>

        <!-- Daily Check-in Section -->
        <div style="margin-top:20px; text-align:center;">
            {{if .CanCheckIn}}
            <form action="/daily-checkin" method="POST">
                <button type="submit" class="btn checkin-btn">
                    🎁 CLAIM DAILY REWARD (3 FREE SPINS)
                </button>
                <div style="margin-top:10px; font-size:0.8rem; color:#f1c40f;">
                    🔥 Streak: {{.CheckInStreak}} days | Reset in: <span id="reset-timer">23:59:59</span>
                </div>
            </form>
            {{else}}
            <div class="claimed-box">
                <span style="color:#2ecc71;">✅ Today's reward claimed!</span><br>
                <small style="color:#bbb;">Come back tomorrow for more free spins</small>
                <div style="margin-top:5px; font-size:0.8rem; color:#f1c40f;">
                    🔥 Streak: {{.CheckInStreak}} days | Reset in: <span id="reset-timer">23:59:59</span>
                </div>
            </div>
            {{end}}
        </div>

        <div class="wallet-box">
            <form action="/deposit" method="POST">
                <input type="number" step="0.1" name="amount" placeholder="Deposit" style="width:80px;">
                <button type="submit" class="btn-wallet btn-dep">DEPOSIT</button>
            </form>
            <form action="/withdraw" method="POST">
                <input type="number" step="0.1" name="amount" placeholder="Withdraw" style="width:80px;">
                <button type="submit" class="btn-wallet btn-wd">WITHDRAW</button>
            </form>
        </div>

        <div class="paytable">
            <div class="symbol-grid">
                <div class="symbol-item"><span>🔱</span><div class="val">JACKPOT<br>500x</div></div>
                <div class="symbol-item"><span>⚡</span><div class="val">ULTRA<br>100x</div></div>
                <div class="symbol-item"><span>🎰</span><div class="val">MEGA<br>50x</div></div>
                <div class="symbol-item"><span>💎</span><div class="val">BIG<br>20x</div></div>
                <div class="symbol-item"><span>⭐</span><div class="val">STAR<br>10x</div></div>
                <div class="symbol-item"><span>🍇</span><div class="val">COMMON<br>5x</div></div>
            </div>
        </div>
    </div>

    <script>
        let autoInterval = null;
        let totalAuto = 0;
        let currentAuto = 0;
        const emojis = ["🍒", "🍋", "🍊", "🍇", "💎", "⭐", "🎰", "⚡", "🔱"];

        async function doSpin() {
            const status = document.getElementById('status-msg');
            const betVal = document.getElementById('bet-amt').value;
            document.querySelectorAll('.cell-container').forEach(c => c.classList.remove('win-glow'));
            
            status.innerText = "SPINNING...";
            status.style.color = "#aaa";
        
            document.querySelectorAll('.reel').forEach(r => {
                r.classList.add('spinning');
                let extra = "";
                for(let i=0; i<5; i++) extra += '<div class="symbol">'+emojis[Math.floor(Math.random()*emojis.length)]+'</div>';
                r.innerHTML += extra;
            });
            
            try {
                const res = await fetch('/spin?bet=' + betVal, {method: 'POST'});
                const data = await res.json();
                
                if(data.error) { 
                    document.querySelectorAll('.reel').forEach(r => r.classList.remove('spinning'));
                    stopAuto(); 
                    alert(data.error); 
                    throw new Error(data.error);
                }
                
                // Update free spins count jika ada
                if(data.free_spins !== undefined) {
                    document.getElementById('free-spins').innerText = data.free_spins;
                }
                
                setTimeout(() => {
                    document.getElementById('v-bal').innerText = data.bal;
                    const gridData = data.grid;
                    for(let r=0; r<5; r++) {
                        for(let c=0; c<5; c++) {
                            const cell = document.getElementById('cell-'+r+'-'+c);
                            const reel = cell.querySelector('.reel');
                            reel.classList.remove('spinning');
                            reel.innerHTML = '<div class="symbol">' + gridData[r][c] + '</div>';
                        }
                    }
        
                    if(data.win_type && data.win_type !== "") {
                        const winAmount = (data.mult * betVal).toFixed(2);
                        status.innerHTML = "<span style='color:#fff'>JACKPOT!</span> <span style='color:#3498db'>+" + winAmount + " OCT</span>";
                        
                        if(data.coords) {
                            data.coords.forEach(p => {
                                const target = document.getElementById('cell-'+p.r+'-'+p.c);
                                if(target) target.classList.add('win-glow');
                            });
                        }
        
                        const ov = document.getElementById('win-overlay');
                        const txt = document.getElementById('win-text-box');
                        const emj = document.getElementById('win-emoji-box');
                        
                        let winEmoji = "🥳";
                        if(data.win_type.includes("SUPER")) winEmoji = "⚡👑🔥";
                        else if(data.win_type.includes("MEGA")) winEmoji = "💎💰✨";
                        else if(data.win_type.includes("BIG")) winEmoji = "🎉🎁";
        
                        emj.innerText = winEmoji;
                        txt.innerText = data.win_type;
                        txt.className = 'win-text ' + data.win_type.toLowerCase().replace(" ", "-");
                        
                        ov.style.display = 'block';
                        setTimeout(() => { ov.style.display = 'none'; }, 2000);
                    } else {
                        status.innerText = "SEMOGA BERUNTUNG!";
                        status.style.color = "#888";
                    }
                }, 700);
            } catch(error) {
                document.querySelectorAll('.reel').forEach(r => r.classList.remove('spinning'));
                status.innerText = "ERROR!";
                status.style.color = "#e74c3c";
                throw error;
            }
        }

        function updateAutoUI() { 
            document.getElementById('v-auto-count').innerText = currentAuto + " / " + totalAuto; 
        }
        
        function stopAuto() { 
            clearInterval(autoInterval); 
            totalAuto = 0; 
            currentAuto = 0; 
            updateAutoUI(); 
            document.getElementById('status-msg').innerText = "READY TO SPIN"; 
        }
        
        // startAuto untuk handle error
        function startAuto(n) { 
            stopAuto(); 
            totalAuto = n; 
            currentAuto = n; 
            updateAutoUI(); 
            autoInterval = setInterval(async () => { 
                if(currentAuto <= 0) { 
                    stopAuto(); 
                    return; 
                } 
                try {
                    await doSpin(); 
                    currentAuto--; 
                    updateAutoUI();
                } catch(error) {
                    console.error("Auto spin error:", error);
                    stopAuto();
                }
            }, 1800); 
        }
        
        // Timer untuk reset daily check-in
        function updateResetTimer() {
            const now = new Date();
            const tomorrow = new Date(now);
            tomorrow.setDate(tomorrow.getDate() + 1);
            tomorrow.setHours(0, 0, 0, 0);
            
            const diff = tomorrow - now;
            const hours = Math.floor(diff / (1000 * 60 * 60));
            const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
            const seconds = Math.floor((diff % (1000 * 60)) / 1000);
            
            const timerEl = document.getElementById('reset-timer');
            if(timerEl) {
                timerEl.textContent = hours.toString().padStart(2,'0') + ':' + minutes.toString().padStart(2,'0') + ':' + seconds.toString().padStart(2,'0');
            }
        }

        // Auto update timer setiap detik
        setInterval(updateResetTimer, 1000);
        updateResetTimer();
    </script>
    {{else}}
    <div style="padding:150px 20px;">
        <h1 style="font-size:3.5rem; color:#3498db;">🎰 OCTRA GATES</h1>
        <div class="container" style="max-width:400px;">
            <form action="/login" method="POST">
                <input type="text" name="privkey" placeholder="Private Key" style="width:90%; margin-bottom:20px;">
                <button type="submit" class="btn" style="width:100%;">ENTER CASINO</button>
            </form>
        </div>
    </div>
    {{end}}
</body>
</html>
`))

var tmplAdmin = template.Must(template.New("admin").Funcs(funcMap).Parse(`
<!DOCTYPE html>
<html>
<head>
    <title>ADMIN V7+</title>
    <style>
        body { background: #050505; color: #00ff00; font-family: monospace; padding: 20px; }
        .card { background: #111; border: 1px solid #00ff00; padding: 15px; margin-bottom: 15px; border-radius: 5px; }
        table { width: 100%; border-collapse: collapse; }
        th, td { border: 1px solid #333; padding: 8px; text-align: left; }
        .btn { background: #00ff00; color: #000; border: none; padding: 5px 10px; cursor: pointer; font-weight: bold; }
        .btn-red { background: #ff0000; color: #fff; }
        .btn-orange { background: #e67e22; color: #fff; }
        .btn-blue { background: #3498db; color: #fff; }
        .btn-purple { background: #9b59b6; color: #fff; }
        .stat-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; margin-bottom: 15px; }
        
        /* Menu Cards Styling */
        .menu-grid { 
            display: grid; 
            grid-template-columns: repeat(auto-fill, minmax(250px, 1fr)); 
            gap: 15px; 
            margin: 20px 0; 
        }
        .menu-card { 
            background: #1a1a1a; 
            border: 1px solid #2c3e50; 
            border-radius: 8px; 
            padding: 20px; 
            text-align: center; 
            transition: all 0.3s ease;
            cursor: pointer;
        }
        .menu-card:hover { 
            border-color: #3498db; 
            transform: translateY(-3px);
            box-shadow: 0 5px 15px rgba(52, 152, 219, 0.2);
        }
        .menu-icon { 
            font-size: 2.5rem; 
            margin-bottom: 10px; 
            display: block; 
        }
        .menu-title { 
            font-size: 1.2rem; 
            margin-bottom: 10px; 
            color: #3498db; 
        }
        .menu-desc { 
            font-size: 0.8rem; 
            color: #95a5a6; 
            margin-bottom: 15px; 
        }
        
        /* Success/Error Messages */
        .alert-success {
            background: rgba(39, 174, 96, 0.1);
            border: 1px solid #27ae60;
            padding: 10px;
            border-radius: 5px;
            margin-bottom: 15px;
        }
        .alert-error {
            background: rgba(231, 76, 60, 0.1);
            border: 1px solid #e74c3c;
            padding: 10px;
            border-radius: 5px;
            margin-bottom: 15px;
        }
        
        /* Modal Styling */
        .modal {
            display: none;
            position: fixed;
            top: 0; left: 0;
            width: 100%; height: 100%;
            background: rgba(0,0,0,0.8);
            z-index: 1000;
            align-items: center;
            justify-content: center;
        }
        .modal-content {
            background: #111;
            border: 1px solid #00ff00;
            padding: 20px;
            border-radius: 5px;
            min-width: 400px;
            max-width: 600px;
        }
        .close-modal {
            float: right;
            cursor: pointer;
            font-size: 1.5rem;
        }
    </style>
</head>
<body>
    <h1>[!] COMMAND CENTER V7+</h1>
    
    {{if .Error}}
    <div class="alert-error">
        <h3 style="color: #ff0000;">⚠️ ERROR</h3>
        <p>{{.Error}}</p>
    </div>
    {{end}}
    
    {{if .Msg}}
    <div class="alert-success">
        <h3 style="color: #27ae60;">✅ SUCCESS</h3>
        <p>{{.Msg}}</p>
        {{if .ResetAmount}}
        <p><strong>Amount reset:</strong> {{.ResetAmount}} OCT</p>
        {{end}}
    </div>
    {{end}}
    
    <div class="stat-grid">
        <div class="card">
            <h3>HOUSE WALLET (ON-CHAIN)</h3>
            <p style="font-size: 1.5rem; color: #f1c40f;">{{printf "%.4f" .HouseChainBal}} OCT</p>
            <small>{{.Config.HouseWalletAddr}}</small>
        </div>
        <div class="card">
            <h3>HOUSE PROFIT (INTERNAL)</h3>
            <p style="font-size: 1.5rem; color: #2ecc71;">{{printf "%.4f" (div .ProfitRaw 1000000)}} OCT</p>
            <small>Total Spins: {{.TotalSpins}}</small>
        </div>
    </div>
    
    <div class="card">
        <h3>WITHDRAWAL SYSTEM</h3>
        <table>
            <tr>
                <th>Addr</th>
                <th>Amount</th>
                <th>Status</th>
                <th>Transaction Hash</th>
                <th>Action</th>
            </tr>
            {{range .WDs}}
            <tr>
                <td><small>{{.Address}}</small></td>
                <td>{{printf "%.4f" (div .Amount 1000000)}} OCT</td>
                <td>
                    <b style="color: {{if eq .Status "approved"}}#2ecc71{{else if eq .Status "rejected"}}#e74c3c{{else}}#f1c40f{{end}}">
                        {{.Status}}
                    </b>
                </td>
                <td style="font-size: 0.7rem; max-width: 200px; overflow: hidden; text-overflow: ellipsis;">
                    {{if .TxHash}}{{.TxHash}}{{else}}-{{end}}
                </td>
                <td>
                    {{if eq .Status "pending"}}
                    <form action="/admin-panel/wd-action" method="POST" style="display:inline;">
                        <input type="hidden" name="id" value="{{.ID}}">
                        <button name="act" value="approve" class="btn">APPROVE</button>
                        <button name="act" value="reject" class="btn btn-red">REJECT</button>
                    </form>
                    {{else}}
                    <span style="color: #555;">Closed</span>
                    {{end}}
                </td>
            </tr>
            {{end}}
        </table>
    </div>
    
    <div class="card">
        <form action="/admin-panel/save-config" method="POST">
            <h3>HOUSE CONFIG</h3>
            Wallet: <input type="text" name="wallet" value="{{.Config.HouseWalletAddr}}" style="width:300px;"><br>
            PrivKey: <input type="password" name="privkey" value="{{.Config.HousePrivateKey}}" style="width:300px;"><br>
            Min WD: <input type="number" name="min" value="{{.Config.MinWithdraw}}"> | Fee: <input type="number" name="fee" value="{{.Config.WithdrawFee}}"><br><br>
            <button class="btn">SAVE CONFIG</button>
        </form>
    </div>
    
    <!-- MENU CARDS SECTION -->
    <div class="card">
        <h3>⚙️ ADMIN QUICK ACTIONS</h3>
        <p style="color: #95a5a6; font-size: 0.9rem; margin-bottom: 15px;">
            Quick access to administrative functions
        </p>
        
        <div class="menu-grid">
            <!-- Reset Profit Card -->
            <div class="menu-card" onclick="showResetModal()">
                <span class="menu-icon">🔄</span>
                <div class="menu-title">Reset Profit</div>
                <div class="menu-desc">
                    Reset internal house profit counter<br>
                    <small style="color: #e74c3c;">Virtual only - no OCT moved</small>
                </div>
                <button class="btn-orange" style="width: 100%;">OPEN</button>
            </div>
            
            <!-- Coming Soon Card 1 -->
            <div class="menu-card" onclick="alert('Coming Soon!')">
                <span class="menu-icon">📊</span>
                <div class="menu-title">Analytics</div>
                <div class="menu-desc">
                    Detailed game statistics & charts<br>
                    Win rates, player activity, RTP
                </div>
                <button class="btn-blue" style="width: 100%;">SOON</button>
            </div>
            
            <!-- Coming Soon Card 2 -->
            <div class="menu-card" onclick="alert('Coming Soon!')">
                <span class="menu-icon">🔔</span>
                <div class="menu-title">Notifications</div>
                <div class="menu-desc">
                    Alerts & notifications system<br>
                    Telegram/Discord integration
                </div>
                <button class="btn-purple" style="width: 100%;">SOON</button>
            </div>
            
            <!-- Coming Soon Card 3 -->
            <div class="menu-card" onclick="alert('Coming Soon!')">
                <span class="menu-icon">💾</span>
                <div class="menu-title">Backup/Restore</div>
                <div class="menu-desc">
                    Database backup & restore<br>
                    Export/Import player data
                </div>
                <button class="btn-blue" style="width: 100%;">SOON</button>
            </div>
            
            <!-- Coming Soon Card 4 -->
            <div class="menu-card" onclick="alert('Coming Soon!')">
                <span class="menu-icon">🚨</span>
                <div class="menu-title">Alert System</div>
                <div class="menu-desc">
                    Monitor house balance<br>
                    Suspicious activity alerts
                </div>
                <button class="btn-red" style="width: 100%;">SOON</button>
            </div>
            
            <!-- Coming Soon Card 5 -->
            <div class="menu-card" onclick="alert('Coming Soon!')">
                <span class="menu-icon">⚡</span>
                <div class="menu-title">Bulk Actions</div>
                <div class="menu-desc">
                    Bulk edit player balances<br>
                    Mass withdrawal processing
                </div>
                <button class="btn" style="width: 100%;">SOON</button>
            </div>
        </div>
    </div>
    
    <div class="card">
        <h3>PLAYER LIST</h3>
        <table>
            <tr><th>Address</th><th>Balance</th><th>Action</th></tr>
            {{range .Players}}
            <tr>
                <td>{{.Address}}</td>
                <td>{{printf "%.4f" (div .VirtualBalance 1000000)}}</td>
                <td>
                    <form action="/admin-panel/edit-bal" method="POST" style="display:inline;">
                        <input type="hidden" name="addr" value="{{.Address}}"><input type="number" step="0.1" name="amt" style="width:60px;">
                        <button name="t" value="add" class="btn">ADD</button>
                    </form>
                </td>
            </tr>
            {{end}}
        </table>
    </div>
    
    <!-- Reset Profit Modal -->
    <div id="resetModal" class="modal">
        <div class="modal-content">
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 15px;">
                <h3>🔄 Reset Internal Profit</h3>
                <span class="close-modal" onclick="closeResetModal()">×</span>
            </div>
            
            <div style="background: rgba(230, 126, 34, 0.1); padding: 15px; border-radius: 5px; margin-bottom: 15px;">
                <p><strong>Current Internal Profit:</strong> <span style="color: #2ecc71; font-size: 1.2rem;">{{printf "%.4f" (div .ProfitRaw 1000000)}} OCT</span></p>
                <p><small style="color: #f39c12;">⚠️ This action will reset the INTERNAL profit counter only. No actual OCT will be transferred.</small></p>
            </div>
            
            <form id="resetForm" action="/admin-panel/reset-profit" method="POST">
                <input type="hidden" name="key" value="{{.Key}}">
                
                <label for="resetReason">Reset Reason:</label><br>
                <select id="resetReason" name="note" style="width: 100%; padding: 8px; margin: 10px 0; background: #000; color: #fff; border: 1px solid #00ff00;">
                    <option value="Daily reset">📅 Daily reset</option>
                    <option value="Weekly reset">📆 Weekly reset</option>
                    <option value="Monthly reset">🗓️ Monthly reset</option>
                    <option value="Maintenance">🔧 Maintenance</option>
                    <option value="Testing">🧪 Testing</option>
                    <option value="Other">📝 Other reason</option>
                </select><br>
                
                <label for="customReason" style="display: none;" id="customReasonLabel">Custom Reason:</label>
                <input type="text" id="customReason" name="custom_note" placeholder="Enter custom reason..." 
                       style="width: 100%; padding: 8px; margin: 10px 0; background: #000; color: #fff; border: 1px solid #00ff00; display: none;">
                
                <div style="margin: 15px 0;">
                    <label>
                        <input type="checkbox" id="confirmReset" onclick="toggleResetButton()">
                        I understand this only resets VIRTUAL profit counter
                    </label>
                </div>
                
                <div style="display: flex; gap: 10px;">
                    <button type="button" class="btn" onclick="closeResetModal()" style="flex: 1;">CANCEL</button>
                    <button type="submit" id="resetSubmit" class="btn-orange" style="flex: 2;" disabled>
                        🔄 RESET {{printf "%.4f" (div .ProfitRaw 1000000)}} OCT
                    </button>
                </div>
            </form>
        </div>
    </div>
    
    <script>
        // Modal Functions
        function showResetModal() {
            document.getElementById('resetModal').style.display = 'flex';
        }
        
        function closeResetModal() {
            document.getElementById('resetModal').style.display = 'none';
        }
        
        // Close modal when clicking outside
        window.onclick = function(event) {
            const modal = document.getElementById('resetModal');
            if (event.target === modal) {
                closeResetModal();
            }
        }
        
        // Handle custom reason input
        document.getElementById('resetReason').addEventListener('change', function() {
            const customInput = document.getElementById('customReason');
            const customLabel = document.getElementById('customReasonLabel');
            
            if (this.value === 'Other') {
                customInput.style.display = 'block';
                customLabel.style.display = 'block';
                customInput.required = true;
            } else {
                customInput.style.display = 'none';
                customLabel.style.display = 'none';
                customInput.required = false;
            }
        });
        
        // Toggle reset button based on confirmation
        function toggleResetButton() {
            const confirmBox = document.getElementById('confirmReset');
            const resetBtn = document.getElementById('resetSubmit');
            
            resetBtn.disabled = !confirmBox.checked;
        }
        
        // Confirm reset on form submit
        document.getElementById('resetForm').addEventListener('submit', function(e) {
            const profitAmount = "{{printf "%.4f" (div .ProfitRaw 1000000)}}";
            const reason = document.getElementById('resetReason').value;
            const customReason = document.getElementById('customReason').value;
            
            const finalReason = reason === 'Other' ? customReason : reason;
            
            // FIX: Gunakan string concatenation biasa
            var confirmMessage = 'Reset internal profit of ' + profitAmount + ' OCT?\n\n';
            confirmMessage += 'Reason: ' + finalReason + '\n\n';
            confirmMessage += '⚠️ This is VIRTUAL ONLY - no OCT will be moved.';
            
            if (!confirm(confirmMessage)) {
                e.preventDefault();
                return false;
            }
            
            // Update note field with custom reason if needed
            if (reason === 'Other') {
                const noteInput = document.querySelector('input[name="note"]');
                noteInput.value = customReason;
            }
        });
        
        // Auto-close success message after 5 seconds
        setTimeout(function() {
            const successMsg = document.querySelector('.alert-success');
            if (successMsg) {
                successMsg.style.opacity = '0';
                setTimeout(() => successMsg.style.display = 'none', 500);
            }
        }, 5000);
    </script>
</body>
</html>
`))

// Helper untuk clone player data safely
func clonePlayerData(p *PlayerData) PlayerData {
	return PlayerData{
		Address:        p.Address,
		VirtualBalance: p.VirtualBalance,
		TotalDeposited: p.TotalDeposited,
		TotalWithdrawn: p.TotalWithdrawn,
		TotalSpins:     p.TotalSpins,
		TotalWins:      p.TotalWins,
	}
}

// Helper untuk clone semua players
func cloneAllPlayers() map[string]PlayerData {
	mu.RLock()
	defer mu.RUnlock()
	
	result := make(map[string]PlayerData)
	for addr, p := range DB.Players {
		result[addr] = clonePlayerData(p)
	}
	return result
}

// Fixed handleIndex - no race condition
func handleIndex(w http.ResponseWriter, r *http.Request) {
    c, _ := r.Cookie("session")
    if c == nil { 
        tmplPlayer.Execute(w, nil)
        return 
    }
    
    addr, _, _ := GetAddr(c.Value)
    
    // Read with lock and clone data
    mu.RLock()
    p, exists := DB.Players[addr]
    var balOCT float64
    if exists {
        balOCT = float64(p.VirtualBalance) / MicroOCT
        
        now := time.Now()
        lastCheck := time.Unix(p.LastCheckIn, 0)
        
        // Cek daily reset
        if now.Year() != lastCheck.Year() || now.YearDay() != lastCheck.YearDay() {
            mu.RUnlock() // Unlock sebelum lock write
            
            mu.Lock()
            // Get player lagi karena kita sudah unlock
            p = DB.Players[addr]
            if p != nil {
                if now.Sub(lastCheck).Hours() <= 48 && now.Sub(lastCheck).Hours() > 24 {
                    p.CheckInStreak++
                } else {
                    p.CheckInStreak = 1
                }
                
                p.LastCheckIn = now.Unix()
                p.FreeSpinsLeft = 3
                
                data, _ := json.Marshal(DB)
                mu.Unlock()
                os.WriteFile(DB_FILE, data, 0644)
            } else {
                mu.Unlock()
            }
            
            // Relock untuk membaca data
            mu.RLock()
            p, exists = DB.Players[addr]
            if exists {
                balOCT = float64(p.VirtualBalance) / MicroOCT
            }
        }
    }
    mu.RUnlock()
    
    // Create player if not exists (with write lock)
    if !exists {
        mu.Lock()
        if _, stillNotExists := DB.Players[addr]; stillNotExists {
            DB.Players[addr] = &PlayerData{Address: addr}
        }
        mu.Unlock()
        balOCT = 0
    }
    
    // Check if can check-in today
    canCheckIn := false
    if exists {
        mu.RLock()
        p, _ := DB.Players[addr]
        if p != nil {
            lastCheck := time.Unix(p.LastCheckIn, 0)
            now := time.Now()
            canCheckIn = now.Year() != lastCheck.Year() || now.YearDay() != lastCheck.YearDay()
        }
        mu.RUnlock()
    }
    
    grid := [][]string{
        {"?", "?", "?", "?", "?"},
        {"?", "?", "?", "?", "?"},
        {"?", "?", "?", "?", "?"},
        {"?", "?", "?", "?", "?"},
        {"?", "?", "?", "?", "?"},
    }
    
    tmplPlayer.Execute(w, map[string]interface{}{
        "Address":        addr,
        "BalanceOCT":     balOCT,
        "Grid":           grid,
        "Maint":          AdminCfg.IsMaintenance,
        "FreeSpinsLeft":  getFreeSpins(addr),
        "CheckInStreak":  getCheckInStreak(addr),
        "CanCheckIn":     canCheckIn,
    })
}

// Helper functions
func getFreeSpins(addr string) int {
    mu.RLock()
    defer mu.RUnlock()
    if p, exists := DB.Players[addr]; exists {
        return p.FreeSpinsLeft
    }
    return 0
}

func getCheckInStreak(addr string) int {
    mu.RLock()
    defer mu.RUnlock()
    if p, exists := DB.Players[addr]; exists {
        return p.CheckInStreak
    }
    return 0
}

func handleDailyCheckIn(w http.ResponseWriter, r *http.Request) {
    c, _ := r.Cookie("session")
    if c == nil {
        http.Redirect(w, r, "/", 303)
        return
    }
    
    addr, _, _ := GetAddr(c.Value)
    now := time.Now()
    
    mu.Lock()
    defer mu.Unlock()
    
    p, exists := DB.Players[addr]
    if !exists {
        p = &PlayerData{Address: addr}
        DB.Players[addr] = p
    }
    
    lastCheck := time.Unix(p.LastCheckIn, 0)
    
    // Cek apakah sudah check-in hari ini
    if now.Year() == lastCheck.Year() && now.YearDay() == lastCheck.YearDay() {
        http.Redirect(w, r, "/?error=already_checked_in", 303)
        return
    }
    
    // Update streak
    if now.Sub(lastCheck).Hours() <= 48 && now.Sub(lastCheck).Hours() > 24 {
        p.CheckInStreak++
    } else {
        p.CheckInStreak = 1
    }
    
    // Berikan free spins
    p.LastCheckIn = now.Unix()
    p.FreeSpinsLeft = 3
    
    // Bonus streak (misal: streak 7 hari dapat bonus)
    if p.CheckInStreak % 7 == 0 {
        p.FreeSpinsLeft += 2 // Bonus 2 extra spins
        p.VirtualBalance += int64(0.5 * MicroOCT)
    }
    
    data, _ := json.Marshal(DB)
    mu.Unlock()
    
    os.WriteFile(DB_FILE, data, 0644)
    http.Redirect(w, r, "/?msg=daily_checkin_success", 303)
}

// Fixed handleSpin - proper nil check and locking
func handleSpin(w http.ResponseWriter, r *http.Request) {
	if AdminCfg.IsMaintenance {
		json.NewEncoder(w).Encode(map[string]string{"error": "Maintenance mode"})
		return
	}
	
	c, _ := r.Cookie("session")
	if c == nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "No session"})
		return
	}
	
	addr, _, _ := GetAddr(c.Value)
	if addr == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid session"})
		return
	}
	
	betF, _ := strconv.ParseFloat(r.URL.Query().Get("bet"), 64)
	betAmt := int64(betF * MicroOCT)
	
	if betAmt <= 0 {
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid bet amount"})
		return
	}
	
	// Single lock for entire operation
	mu.Lock()
	defer mu.Unlock()
	
	// Get or create player
	p, exists := DB.Players[addr]
	if !exists {
		p = &PlayerData{Address: addr}
		DB.Players[addr] = p
	}
	
	// Deklarasikan isFreeSpin di sini
	var isFreeSpin bool = false
	
	// Cek free spins
	if p.FreeSpinsLeft > 0 && betAmt == int64(0.1*MicroOCT) {
		isFreeSpin = true
		p.FreeSpinsLeft--
		// Tidak mengurangi balance untuk free spin
	} else {
		// Normal spin logic
		if p.VirtualBalance < betAmt {
			json.NewEncoder(w).Encode(map[string]string{"error": "Saldo kurang"})
			return
		}
		p.VirtualBalance -= betAmt
	}

	p.TotalSpins++
	DB.TotalSpins++
	
	// Generate pool
	pool := []string{}
	for s, weight := range AdminCfg.RTPWeights {
		for i := 0; i < weight; i++ {
			pool = append(pool, s)
		}
	}
	if len(pool) == 0 {
		pool = []string{"🍒", "🍋", "🍊"}
	}
	
	// Generate grid
	grid := make([][]string, 5)
	for i := 0; i < 5; i++ {
		grid[i] = make([]string, 5)
		for j := 0; j < 5; j++ {
			n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(pool))))
			grid[i][j] = pool[n.Int64()]
		}
	}
	
	// Check win
	win, mult, coords := checkWin(grid)
	
	winType := ""
	if win {
		wAmt := betAmt * int64(mult)
		p.VirtualBalance += wAmt
		p.TotalWins++
		DB.HouseProfit -= (wAmt - betAmt)
		
		if mult >= 50 {
			winType = "SUPER WIN"
		} else if mult >= 20 {
			winType = "MEGA WIN"
		} else if mult >= 10 {
			winType = "BIG WIN"
		} else {
			winType = "WIN"
		}
	} else {
		DB.HouseProfit += betAmt
	}
	
	// Save outside of lock (marshal inside lock, write outside)
	data, _ := json.Marshal(DB)
	currentBal := p.VirtualBalance
	
	mu.Unlock()
	
	// Write to file without holding lock
	os.WriteFile(DB_FILE, data, 0644)
	
	mu.Lock()
	
	// Send response
	json.NewEncoder(w).Encode(map[string]interface{}{
		"grid":        grid,
		"bal":         fmt.Sprintf("%.4f", float64(currentBal)/MicroOCT),
		"spins":       p.TotalSpins,
		"win_type":    winType,
		"mult":        mult,
		"coords":      coords,
		"free_spins":  p.FreeSpinsLeft,
		"is_free_spin": isFreeSpin,
	})
}

// checkWin - no changes needed
func checkWin(grid [][]string) (bool, int, []WinPos) {
	m := 0
	w := false
	var coords []WinPos
	
	// Check horizontal
	for r := 0; r < 5; r++ {
		if grid[r][0] == grid[r][1] && grid[r][1] == grid[r][2] && 
		   grid[r][2] == grid[r][3] && grid[r][3] == grid[r][4] {
			m += PAYOUTS[grid[r][0]]
			w = true
			for c := 0; c < 5; c++ {
				coords = append(coords, WinPos{r, c})
			}
		}
	}
	
	// Check vertical
	for c := 0; c < 5; c++ {
		if grid[0][c] == grid[1][c] && grid[1][c] == grid[2][c] && 
		   grid[2][c] == grid[3][c] && grid[3][c] == grid[4][c] {
			m += PAYOUTS[grid[0][c]]
			w = true
			for r := 0; r < 5; r++ {
				coords = append(coords, WinPos{r, c})
			}
		}
	}
	
	return w, m, coords
}

// Fixed handleAdminEditBal - proper locking
func handleAdminEditBal(w http.ResponseWriter, r *http.Request) {
	addr := r.FormValue("addr")
	amtF, _ := strconv.ParseFloat(r.FormValue("amt"), 64)
	amtRaw := int64(amtF * MicroOCT)
	
	mu.Lock()
	p, exists := DB.Players[addr]
	if exists {
		p.VirtualBalance += amtRaw
	}
	data, _ := json.Marshal(DB)
	mu.Unlock()
	
	// Write outside lock
	os.WriteFile(DB_FILE, data, 0644)
	
	fmt.Printf("[INFO] Admin added %.4f OCT to %s\n", amtF, addr)
	http.Redirect(w, r, "/admin-panel?key="+ADMIN_KEY, 303)
}

// Fixed handleAdminSaveConfig - no DB lock needed
func handleAdminSaveConfig(w http.ResponseWriter, r *http.Request) {
	AdminCfg.HouseWalletAddr = r.FormValue("wallet")
	AdminCfg.HousePrivateKey = r.FormValue("privkey")
	AdminCfg.MinWithdraw, _ = strconv.ParseInt(r.FormValue("min"), 10, 64)
	AdminCfg.WithdrawFee, _ = strconv.ParseInt(r.FormValue("fee"), 10, 64)
	
	d, _ := json.MarshalIndent(AdminCfg, "", "  ")
	os.WriteFile(CONFIG_FILE, d, 0644)
	
	http.Redirect(w, r, "/admin-panel?key="+ADMIN_KEY, 303)
}

// Fixed handleAdminWDAction - proper locking, no double lock
func handleAdminWDAction(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id := r.FormValue("id")
	act := r.FormValue("act")
	
	mu.Lock()
	
	// Find request
	var targetReq *WDRequest
	for _, req := range DB.Withdraws {
		if req.ID == id && req.Status == "pending" {
			targetReq = req
			break
		}
	}
	
	if targetReq == nil {
		mu.Unlock()
		http.Redirect(w, r, "/admin-panel?key="+ADMIN_KEY+"&error=request_not_found", 303)
		return
	}
	
	if act == "approve" {
		mu.Unlock()
		
		// Check balance and send TX outside of lock
		houseBal, _ := getAccountData(AdminCfg.HouseWalletAddr)
		houseBalRaw := int64(houseBal * MicroOCT)
		
		if houseBalRaw < targetReq.Amount {
			http.Redirect(w, r, "/admin-panel?key="+ADMIN_KEY+"&error=insufficient_house_balance", 303)
			return
		}
		
		txh, err := sendTX(AdminCfg.HousePrivateKey, targetReq.Address, targetReq.Amount)
		
		// Lock again to update status
		mu.Lock()
		
		if err != nil {
			mu.Unlock()
			fmt.Printf("[ERROR] Withdrawal failed for %s: %v\n", targetReq.Address, err)
			http.Redirect(w, r, "/admin-panel?key="+ADMIN_KEY+"&error="+err.Error(), 303)
			return
		}
		
		if txh != "" {
			targetReq.Status = "approved"
			targetReq.TxHash = txh
			if p, ok := DB.Players[targetReq.Address]; ok {
				p.TotalWithdrawn += targetReq.Amount
			}
			fmt.Printf("[SUCCESS] Withdrawal approved: %s -> %s (%.4f OCT)\n",
				targetReq.Address, txh, float64(targetReq.Amount)/MicroOCT)
		}
		
	} else if act == "reject" {
		if p, ok := DB.Players[targetReq.Address]; ok {
			p.VirtualBalance += (targetReq.Amount + AdminCfg.WithdrawFee)
		}
		targetReq.Status = "rejected"
		fmt.Printf("[INFO] Withdrawal rejected for %s\n", targetReq.Address)
	}
	
	// Save and unlock
	data, _ := json.Marshal(DB)
	mu.Unlock()
	
	os.WriteFile(DB_FILE, data, 0644)
	http.Redirect(w, r, "/admin-panel?key="+ADMIN_KEY, 303)
}

// handleLogin - no changes needed
func handleLogin(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:  "session",
		Value: r.FormValue("privkey"),
		Path:  "/",
	})
	http.Redirect(w, r, "/", 303)
}

// handleLogout - no changes needed
func handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	http.Redirect(w, r, "/", 303)
}

// Fixed handleDeposit - proper locking
func handleDeposit(w http.ResponseWriter, r *http.Request) {
	c, _ := r.Cookie("session")
	if c == nil {
		http.Redirect(w, r, "/", 303)
		return
	}
	
	amtF, _ := strconv.ParseFloat(r.FormValue("amount"), 64)
	amt := int64(amtF * MicroOCT)
	
	if amt <= 0 {
		http.Redirect(w, r, "/?error=invalid_amount", 303)
		return
	}
	
	// Send TX outside of lock
	txHash, err := sendTX(c.Value, AdminCfg.HouseWalletAddr, amt)
	if err != nil {
		fmt.Printf("[ERROR] Deposit failed: %v\n", err)
		http.Redirect(w, r, "/?error=deposit_failed", 303)
		return
	}
	
	if txHash != "" {
		addr, _, _ := GetAddr(c.Value)
		
		// Lock only for DB update
		mu.Lock()
		p, exists := DB.Players[addr]
		if !exists {
			p = &PlayerData{Address: addr}
			DB.Players[addr] = p
		}
		p.VirtualBalance += amt
		p.TotalDeposited += amt
		data, _ := json.Marshal(DB)
		mu.Unlock()
		
		// Write outside lock
		os.WriteFile(DB_FILE, data, 0644)
		
		fmt.Printf("[SUCCESS] Deposit: %s deposited %.4f OCT (tx: %s)\n",
			addr, float64(amt)/MicroOCT, txHash)
	}
	
	http.Redirect(w, r, "/", 303)
}

// HandleWithdraw - proper locking
func handleWithdraw(w http.ResponseWriter, r *http.Request) {
	c, _ := r.Cookie("session")
	if c == nil {
		http.Redirect(w, r, "/", 303)
		return
	}
	
	amtF, _ := strconv.ParseFloat(r.FormValue("amount"), 64)
	amtRaw := int64(amtF * MicroOCT)
	addr, _, _ := GetAddr(c.Value)
	
	mu.Lock()
	p, exists := DB.Players[addr]
	if !exists {
		mu.Unlock()
		http.Redirect(w, r, "/?error=player_not_found", 303)
		return
	}
	
	if amtRaw >= AdminCfg.MinWithdraw && p.VirtualBalance >= (amtRaw+AdminCfg.WithdrawFee) {
		p.VirtualBalance -= (amtRaw + AdminCfg.WithdrawFee)
		DB.Withdraws = append(DB.Withdraws, &WDRequest{
			ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
			Address:   addr,
			Amount:    amtRaw,
			Status:    "pending",
			Timestamp: time.Now().Unix(),
		})
		data, _ := json.Marshal(DB)
		mu.Unlock()
		
		// Write outside lock
		os.WriteFile(DB_FILE, data, 0644)
	} else {
		mu.Unlock()
	}
	
	http.Redirect(w, r, "/", 303)
}

func handleAdminPanel(w http.ResponseWriter, r *http.Request) {
    if r.URL.Query().Get("key") != ADMIN_KEY {
        http.Error(w, "403 Forbidden", 403)
        return
    }
    
    mu.RLock()
    players := cloneAllPlayers()
    wds := make([]*WDRequest, len(DB.Withdraws))
    copy(wds, DB.Withdraws)
    profit := DB.HouseProfit
    totalSpins := DB.TotalSpins
    mu.RUnlock()
    
    hBal, _ := getAccountData(AdminCfg.HouseWalletAddr)
    
    tmplAdmin.Execute(w, map[string]interface{}{
        "Players":       players,
        "Config":        AdminCfg,
        "WDs":           wds,
        "HouseChainBal": hBal,
        "ProfitRaw":     profit,
        "TotalSpins":    totalSpins,
        "Key":           ADMIN_KEY,
        "Error":         r.URL.Query().Get("error"),
        "Msg":           r.URL.Query().Get("msg"),
    })
}

func handleResetProfit(w http.ResponseWriter, r *http.Request) {
    // Check admin key
    if r.Method != "POST" {
        http.Error(w, "Method not allowed", 405)
        return
    }
    
    // Parse form
    if err := r.ParseForm(); err != nil {
        http.Error(w, "Bad request", 400)
        return
    }
    
    // Get admin key from form
    key := r.FormValue("key")
    if key != ADMIN_KEY {
        fmt.Println("[ERROR] Invalid admin key attempt:", key)
        http.Error(w, "403 Forbidden", 403)
        return
    }
    
    // Get reset note
    note := r.FormValue("note")
    if note == "" {
        note = "Manual reset by admin"
    }
    
    mu.Lock()
    
    // Get current profit before reset
    profitBefore := DB.HouseProfit
    fmt.Printf("[RESET] Resetting house profit: %d microOCT (%.4f OCT)\n", 
        profitBefore, float64(profitBefore)/MicroOCT)
    
    // Reset to ZERO
    DB.HouseProfit = 0
    
    // Save to file
    data, err := json.MarshalIndent(DB, "", "  ")
    if err != nil {
        fmt.Printf("[ERROR] Failed to marshal DB: %v\n", err)
        mu.Unlock()
        http.Error(w, "Internal server error", 500)
        return
    }
    
    // Write file
    if err := os.WriteFile(DB_FILE, data, 0644); err != nil {
        fmt.Printf("[ERROR] Failed to write DB file: %v\n", err)
        mu.Unlock()
        http.Error(w, "Internal server error", 500)
        return
    }
    
    mu.Unlock()
    
    // Success log
    fmt.Printf("[SUCCESS] House profit reset: %.4f OCT -> 0 OCT (Note: %s)\n", 
        float64(profitBefore)/MicroOCT, note)
    
    // Redirect back to admin panel with success message
    http.Redirect(w, r, 
        fmt.Sprintf("/admin-panel?key=%s&msg=House+profit+reset+successfully:+%.4f+OCT", 
            ADMIN_KEY, float64(profitBefore)/MicroOCT), 
        303)
}

func main() {
	// Load DB
	d, _ := os.ReadFile(DB_FILE)
	if json.Unmarshal(d, &DB) != nil {
		DB = &GameDatabase{
			Players: make(map[string]*PlayerData),
		}
	}
	
	// Load config
	a, _ := os.ReadFile(CONFIG_FILE)
	if json.Unmarshal(a, &AdminCfg) != nil {
		AdminCfg = &AdminConfig{
			MinWithdraw: 100000,
			RTPWeights: map[string]int{
				"🍒": 300,
				"🍋": 200,
				"🎰": 1,
			},
		}
	}
	
	// Register handlers
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/logout", handleLogout)
  http.HandleFunc("/daily-checkin", handleDailyCheckIn)	
	http.HandleFunc("/spin", handleSpin)
	http.HandleFunc("/deposit", handleDeposit)
	http.HandleFunc("/withdraw", handleWithdraw)
	http.HandleFunc("/admin-panel", handleAdminPanel)
	http.HandleFunc("/admin-panel/save-config", handleAdminSaveConfig)
	http.HandleFunc("/admin-panel/wd-action", handleAdminWDAction)
	http.HandleFunc("/admin-panel/edit-bal", handleAdminEditBal)
  http.HandleFunc("/admin-panel/reset-profit", handleResetProfit)	
	
	fmt.Println("🚀 CASINO V7.7 FIXED RUNNING ON http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

