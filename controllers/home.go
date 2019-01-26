package controllers

import (
	"net/http"
	"strings"
	"time"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/validation"
	"github.com/gorilla/websocket"
)

// HomeController : controller
type HomeController struct {
	beego.Controller
}

// RequestDetails form details
type RequestDetails struct {
	URL      string `form:"url" valid:"Required"`
	Headers  string `form:"headers"`
	Method   string `form:"method" valid:"Required"`
	Payload  string `form:"payload"`
	Users    int    `form:"users" valid:"Required"`
	Duration int    `form:"duration"`
	Format   string `form:"format"`
}

// Get request
func (c *HomeController) Get() {
	log := logs.NewLogger()
	log.SetLogger(logs.AdapterConsole)

	flash := beego.ReadFromRequest(&c.Controller)
	if _, ok := flash.Data["notice"]; ok {
		// Display settings successful
		c.Data["notice"] = true
	} else if _, ok = flash.Data["error"]; ok {
		c.Data["error"] = true
	}

	c.Data["slaves"] = slaves
	if !running {
		c.TplName = "home.tpl"
	} else {
		c.Data["users"] = users
		c.TplName = "burst.tpl"
	}
}

// Post request
func (c *HomeController) Post() {
	log := logs.NewLogger()
	log.SetLogger(logs.AdapterConsole)

	command := c.GetString("command")

	if command == "start" {
		r := &RequestDetails{}
		c.ParseForm(r)
		flash := beego.NewFlash()

		valid := validation.Validation{}
		isValid, err := valid.Valid(r)

		if !isValid {
			for _, e := range valid.Errors {
				flash.Error("%#v %#v", e.Key, e.Message)
				break
			}
			c.Redirect("/", 302)
		}

		if err == nil {
			quit = make(chan bool)
			var headerList []string

			if len(strings.TrimSpace(r.Headers)) > 0 {
				headerList = strings.Split(r.Headers, ";")
			}

			running = true
			users = r.Users
			batchSize = users * 10

			if slaves == 0 {
				for i := 0; i < r.Users; i++ {
					log.Debug("Starting user %#v", i+1)
					go func() {
						for {
							select {
							case <-quit:
								log.Debug("Returning from go routine")
								return
							default:
								meteorBurst(r.URL, r.Method, r.Payload, headerList)
							}
						}
					}()
				}
			} else {
				runOnSlaves(r, headerList)
			}
		} else {
			flash.Error("%#v", err.Error())
		}

		if r.Duration > 0 {
			if r.Format == "seconds" {
				timer = time.NewTimer(time.Second * time.Duration(r.Duration))
			} else if r.Format == "minutes" {
				timer = time.NewTimer(time.Minute * time.Duration(r.Duration))
			}
			go timeKeeper(r.Duration, r.Format)
		}

		setStartTime(time.Now().UnixNano() / int64(time.Millisecond))

		flash.Store(&c.Controller)
		c.Redirect("/", 302)

	} else if command == "stop" {
		if quit != nil {
			stopEverything()
		}
		c.Data["json"] = "{'stopped': true}"
		c.ServeJSON()
	}
}

// Join creates a new websocket object for a new client and adds to subsriber list
func (c *HomeController) Join() {
	ws, err := websocket.Upgrade(c.Ctx.ResponseWriter, c.Ctx.Request, nil, 1024, 1024)

	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(c.Ctx.ResponseWriter, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		beego.Error("Cannot setup WebSocket connection:", err)
		return
	}

	Join(ws)
	c.Data["success"] = true
	c.ServeJSON()
}
