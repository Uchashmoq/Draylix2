package ui

import (
	"Draylix2/network"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"strings"
	"sync"
	"time"
)

const (
	AppTitle = "Draylix 2.0"
)

type ClientTUI struct {
	app          *tview.Application
	LogChan      chan string
	UpChan       chan int64
	DownChan     chan int64
	trafficSum   int64
	trafficMutex sync.Mutex
	proxyOn      bool

	uploadSpeed   *tview.TextView
	downloadSpeed *tview.TextView
	address       *tview.TextView
	currentNode   *tview.TextView
	logView       *tview.TextView
	traffic       *tview.TextView

	ProxyControl func(on bool)
	proxyHint    *tview.TextView
}

func (ct *ClientTUI) SetAddress(addr string) {
	ct.submitDraw(func() {
		ct.address.SetText(addr)
	})
}

func (ct *ClientTUI) SetNode(node string) {
	ct.submitDraw(func() {
		ct.currentNode.SetText(node)
	})
}

func NewClientTUI() *ClientTUI {
	ui := &ClientTUI{
		LogChan:  make(chan string, 16),
		UpChan:   make(chan int64, 16),
		DownChan: make(chan int64, 16),
	}

	ui.initTUI()
	return ui
}
func (ct *ClientTUI) Run() error {
	ct.updatingSpeedTraffic()
	return ct.app.Run()
}

func (ct *ClientTUI) initTUI() {
	ct.app = tview.NewApplication()
	top := ct.initTop()
	mid := ct.initMid()
	bott := ct.initBott()

	flex := tview.NewFlex()
	flex.SetDirection(tview.FlexRow)
	flex.AddItem(top, 0, 1, false)
	flex.AddItem(mid, 0, 4, false)
	flex.AddItem(bott, 0, 1, true)
	ct.app.SetRoot(flex, true)
}

func initFlexKV(k, v string) (*tview.TextView, *tview.TextView, *tview.Flex) {
	viewK := tview.NewTextView()
	viewK.SetText(k)
	viewK.SetDynamicColors(true)
	viewK.SetTextAlign(tview.AlignLeft)

	viewV := tview.NewTextView()
	viewV.SetText(v)
	viewV.SetDynamicColors(true)
	viewV.SetTextAlign(tview.AlignRight)

	flex := tview.NewFlex().SetDirection(tview.FlexColumn)
	flex.AddItem(viewK, 0, 1, false)
	flex.AddItem(viewV, 0, 1, false)

	return viewK, viewV, flex
}

func (ct *ClientTUI) initTop() *tview.Flex {
	titleAndSpeedFlex := ct.initTitleAndSpeed()

	_, node, nodeFlex := initFlexKV("Node:", "")
	ct.currentNode = node

	_, address, addressFlex := initFlexKV("Address:", "")
	ct.address = address

	topFlex := tview.NewFlex()
	topFlex.SetDirection(tview.FlexRow)
	topFlex.SetBorder(true)
	topFlex.AddItem(titleAndSpeedFlex, 0, 1, false)
	topFlex.AddItem(nodeFlex, 0, 1, false)
	topFlex.AddItem(addressFlex, 0, 1, false)
	return topFlex
}

func (ct *ClientTUI) initTitleAndSpeed() *tview.Flex {
	title := tview.NewTextView()
	title.SetText(AppTitle)
	title.SetDynamicColors(true)
	title.SetTextAlign(tview.AlignLeft)

	up := tview.NewTextView()
	up.SetText("↑ 0 B/S")
	up.SetDynamicColors(true)
	up.SetTextAlign(tview.AlignLeft)
	ct.uploadSpeed = up

	down := tview.NewTextView()
	down.SetText("↓ 0 B/S")
	down.SetDynamicColors(true)
	down.SetTextAlign(tview.AlignRight)
	ct.downloadSpeed = down

	speedFlex := tview.NewFlex()
	speedFlex.SetDirection(tview.FlexColumn)
	speedFlex.AddItem(up, 0, 1, false)
	speedFlex.AddItem(down, 0, 1, false)

	titleSpeedFlex := tview.NewFlex()
	titleSpeedFlex.SetDirection(tview.FlexColumn)
	titleSpeedFlex.AddItem(title, 0, 1, false)
	titleSpeedFlex.AddItem(nil, 0, 1, false)
	titleSpeedFlex.AddItem(speedFlex, 0, 1, false)

	return titleSpeedFlex
}

func (ct *ClientTUI) initMid() *tview.Flex {
	logView := tview.NewTextView()
	logView.SetTextAlign(tview.AlignLeft)
	logView.SetDynamicColors(true)
	logView.SetBorderPadding(1, 1, 0, 0)
	logView.SetScrollable(true)

	ct.logView = logView
	midFlex := tview.NewFlex()
	midFlex.SetDirection(tview.FlexRow)
	midFlex.SetBorder(true)
	midFlex.AddItem(logView, 0, 1, false)
	return midFlex
}

func (ct *ClientTUI) submitDraw(f func()) {
	go func() {
		ct.app.QueueUpdateDraw(func() {
			f()
		})
	}()
}

func (ct *ClientTUI) Log(msg string) {
	text := ct.logView.GetText(true)
	msg = strings.TrimRight(msg, "\n")
	text = text + msg + "\n"
	ct.submitDraw(func() {
		ct.logView.SetText(text)
		ct.logView.ScrollToEnd()
	})
}

func (ct *ClientTUI) SetUpSpeed(s int64) {
	speed := speedFormat(true, s)
	ct.submitDraw(func() {
		ct.uploadSpeed.SetText(speed)
	})
}

func (ct *ClientTUI) SetDownSpeed(s int64) {
	speed := speedFormat(false, s)
	ct.submitDraw(func() {
		ct.downloadSpeed.SetText(speed)
	})
}

func speedFormat(up bool, spd int64) string {
	spdStr := network.BytesFormat(spd) + "/s"
	if up {
		spdStr = "↑ " + spdStr
	} else {
		spdStr = "↓ " + spdStr
	}
	return spdStr
}

func (ct *ClientTUI) updateUpSpeed() {
	ticker := time.NewTicker(1 * time.Second)
	data := int64(0)
	for {
		select {
		case d := <-ct.UpChan:
			data += d
		case <-ticker.C:
			ct.trafficMutex.Lock()
			ct.trafficSum += data
			ct.trafficMutex.Unlock()
			ct.SetUpSpeed(data)
			data = 0
		}
	}
}

func (ct *ClientTUI) updateDownSpeed() {
	ticker := time.NewTicker(1 * time.Second)
	data := int64(0)
	for {
		select {
		case d := <-ct.DownChan:
			data += d
		case <-ticker.C:
			ct.trafficMutex.Lock()
			ct.trafficSum += data
			ct.trafficMutex.Unlock()
			ct.SetDownSpeed(data)
			data = 0
		}
	}
}

func (ct *ClientTUI) updatingSpeedTraffic() {
	go ct.updateUpSpeed()
	go ct.updateDownSpeed()
	go ct.updateTraffic()
}

func (ct *ClientTUI) updateTraffic() {
	for {
		time.Sleep(1 * time.Second)
		ct.SetTraffic(ct.trafficSum)
	}

}

func (ct *ClientTUI) SetTraffic(traffic int64) {
	data := network.BytesFormat(traffic)
	ct.submitDraw(func() {
		ct.traffic.SetText(data)
	})

}

func (ct *ClientTUI) initBott() *tview.Flex {
	menu := ct.initMenu()
	traffic := tview.NewTextView().SetText("0 B")
	traffic.SetTextAlign(tview.AlignRight)
	ct.traffic = traffic

	bott := tview.NewFlex().SetDirection(tview.FlexColumn)
	bott.SetBorder(true)
	bott.AddItem(menu, 0, 10, true)
	bott.AddItem(traffic, 0, 20, false)

	return bott
}

func (ct *ClientTUI) initMenu() *tview.Flex {
	_, proxyHint, proxyFlex := initFlexKV("[Ctrl+D]:", "System Proxy On")
	ct.proxyHint = proxyHint
	menu := tview.NewFlex().SetDirection(tview.FlexRow)
	menu.AddItem(proxyFlex, 0, 1, true)

	ct.initMenuEvent(menu)
	return menu
}

func (ct *ClientTUI) initMenuEvent(menu *tview.Flex) {
	menu.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlD:
			ct.SetProxy(!ct.proxyOn)
		}
		return event
	})
}

func (ct *ClientTUI) SetProxy(status bool) {
	if ct.ProxyControl != nil {
		ct.ProxyControl(status)
	}
	ct.submitDraw(func() {
		if status {
			ct.proxyHint.SetText("System Proxy Off")
		} else {
			ct.proxyHint.SetText("System Proxy On")
		}
	})
	ct.proxyOn = status
}
