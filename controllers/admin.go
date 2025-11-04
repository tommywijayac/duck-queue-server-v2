package controllers

import (
	"net/http"
	"os/exec"

	"github.com/beego/beego/v2/server/web"
)

type AdminController struct {
	web.Controller
}

func (c *AdminController) ExitDispenserApp() {
	type Request struct {
		PIN string `json:"pin"`
	}

	var req Request
	if err := c.BindJSON(&req); err != nil {
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Data["json"] = map[string]string{
			"error":       "Invalid input",
			"dev_message": err.Error(),
		}
		c.ServeJSON()
		return
	}

	wantPIN, err := web.AppConfig.String("app::admin_pin")
	if err != nil {
		wantPIN = ""
	}

	if wantPIN != req.PIN {
		c.Ctx.Output.SetStatus(http.StatusUnauthorized)
		c.Data["json"] = map[string]string{
			"error": "Unauthorized",
		}
		c.ServeJSON()
		return
	}

	cmd := exec.Command("systemctl", "stop", "dispenser")
	_, err = cmd.CombinedOutput()
	if err != nil {
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{
			"error":       "Fail to stop application",
			"dev_message": err.Error(),
		}
		c.ServeJSON()
		return
	}

	c.Ctx.Output.SetStatus(http.StatusOK)
	c.Data["json"] = map[string]interface{}{
		"message": "Dispenser stopped successfully",
	}
	c.ServeJSON()
}
