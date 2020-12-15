package healthcheck

import (
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"github.com/hawell/logger"
	"github.com/hawell/z42/internal/storage"
	"github.com/hawell/z42/internal/types"
	"github.com/hawell/z42/pkg/workerpool"
	"github.com/pkg/errors"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Healthcheck struct {
	Enable             bool
	maxRequests        int
	maxPendingRequests int
	updateInterval     time.Duration
	checkInterval      time.Duration
	redisData          *storage.DataHandler
	redisStat          *storage.StatHandler
	logger             *logger.EventLogger
	lastUpdate         time.Time
	dispatcher         *workerpool.Dispatcher
	quit               chan struct{}
	quitWG             sync.WaitGroup
}

func HandleHealthCheck(h *Healthcheck) workerpool.JobHandler {
	return func(worker *workerpool.Worker, job workerpool.Job) {
		item := job.(*types.HealthCheckItem)
		// logger.Default.Debugf("item %v received", item)
		var err error
		switch item.Protocol {
		case "http", "https":
			timeout := time.Duration(item.Timeout) * time.Millisecond
			url := item.Protocol + "://" + item.Ip + item.Uri
			err = httpCheck(url, item.Host, timeout)
		case "ping", "icmp":
			err = pingCheck(item.Ip, time.Duration(item.Timeout)*time.Millisecond)
			logger.Default.Error("@@@@@@@@@@@@@@ ", item.Ip, " : result : ", err)
		default:
			err = errors.New(fmt.Sprintf("invalid protocol : %s used for %s:%d", item.Protocol, item.Ip, item.Port))
			logger.Default.Error(err)
		}
		item.Error = err
		if err == nil {
			statusUp(item)
		} else {
			statusDown(item)
		}
		item.LastCheck = time.Now()
		h.redisStat.SetHealthcheckItem(item)
		h.logHealthcheck(item)
	}
}

func httpCheck(url string, host string, timeout time.Duration) error {
	tr := &http.Transport{
		MaxIdleConnsPerHost: 1024,
		TLSHandshakeTimeout: 0 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         host,
		},
	}
	client := &http.Client{
		Timeout:   timeout,
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		logger.Default.Errorf("invalid request, host:%s, url:%s : %s", host, url, err)
		return err
	}
	req.Host = strings.TrimRight(host, ".")
	resp, err := client.Do(req)
	if err != nil {
		logger.Default.Errorf("request failed, host:%s, url:%s : %s", host, url, err)
		return err
	}
	switch resp.StatusCode {
	case http.StatusOK, http.StatusFound, http.StatusMovedPermanently:
		return nil
	default:
		return errors.New(fmt.Sprintf("invalid http status code : %d", resp.StatusCode))
	}
}

// FIXME: ping check is not working properly
func pingCheck(ip string, timeout time.Duration) error {
	c, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return err
	}
	c.SetDeadline(time.Now().Add(timeout))
	defer c.Close()

	id := int(binary.BigEndian.Uint32(net.ParseIP(ip)))
	wm := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID:   id,
			Data: []byte("HELLO-R-U-THERE"),
		},
	}
	wb, err := wm.Marshal(nil)
	if err != nil {
		return err
	}
	if _, err := c.WriteTo(wb, &net.IPAddr{IP: net.ParseIP(ip)}); err != nil {
		return err
	}

	rb := make([]byte, 1500)
	n, _, err := c.ReadFrom(rb)
	if err != nil {
		return err
	}
	rm, err := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), rb[:n])
	if err != nil {
		return err
	}
	switch rm.Type {
	case ipv4.ICMPTypeEchoReply:
		logger.Default.Error("@@@@@@@@@@@@ code = ", rm.Code)
		return nil
	default:
		return errors.New(fmt.Sprintf("got %+v; want echo reply", rm))
	}
}

type HealthcheckConfig struct {
	Enable             bool             `json:"enable"`
	MaxRequests        int              `json:"max_requests"`
	MaxPendingRequests int              `json:"max_pending_requests"`
	UpdateInterval     int              `json:"update_interval"`
	CheckInterval      int              `json:"check_interval"`
	Log                logger.LogConfig `json:"log"`
}

func NewHealthcheck(config *HealthcheckConfig, redisData *storage.DataHandler, redisStat *storage.StatHandler) *Healthcheck {
	h := &Healthcheck{
		Enable:             config.Enable,
		maxRequests:        config.MaxRequests,
		maxPendingRequests: config.MaxPendingRequests,
		updateInterval:     time.Duration(config.UpdateInterval) * time.Second,
		checkInterval:      time.Duration(config.CheckInterval) * time.Second,
	}

	if h.Enable {

		h.redisData = redisData
		h.redisStat = redisStat
		h.dispatcher = workerpool.NewDispatcher(config.MaxPendingRequests, config.MaxRequests)
		for i := 0; i < config.MaxRequests; i++ {
			h.dispatcher.AddWorker(HandleHealthCheck(h))
		}
		h.logger = logger.NewLogger(&config.Log, nil)
		h.quit = make(chan struct{}, 1)
	}

	return h
}

func (h *Healthcheck) ShutDown() {
	if !h.Enable {
		return
	}
	// fmt.Println("healthcheck : stopping")
	h.dispatcher.Stop()
	h.quitWG.Add(2) // one for h.dispatcher.Start(), another for h.Transfer()
	close(h.quit)
	h.quitWG.Wait()
	// fmt.Println("healthcheck : stopped")
}

func (h *Healthcheck) getDomainId(zone string) string {
	cfg, err := h.redisData.GetZoneConfig(zone)
	if err != nil {
		return ""
	}
	return cfg.DomainId
}

func (h *Healthcheck) Start() {
	if !h.Enable {
		return
	}
	h.dispatcher.Run()

	go h.Transfer()

	ticker := time.NewTicker(h.checkInterval)
	for {
		itemKeys, err := h.redisStat.GetActiveHealthcheckItems()
		if err != nil {
			logger.Default.Errorf("cannot load healthcheck active items : %s", err)
		}
		select {
		case <-h.quit:
			ticker.Stop()
			h.quitWG.Done()
			return
		case <-ticker.C:
			for _, itemKey := range itemKeys {
				item, err := h.redisStat.GetHealthcheckItem(itemKey)
				if err == nil {
					if time.Since(item.LastCheck) > h.checkInterval {
						h.dispatcher.Queue(item)
					}
				}
			}
		}
	}

}

func (h *Healthcheck) logHealthcheck(item *types.HealthCheckItem) {
	data := map[string]interface{}{
		"ip":          item.Ip,
		"port":        item.Port,
		"host":        item.Host,
		"domain_uuid": item.DomainId,
		"uri":         item.Uri,
		"status":      item.Status,
		"log_type":    "healthcheck",
	}
	if item.Error == nil {
		data["error"] = ""
	} else {
		data["error"] = item.Error.Error()
	}

	h.logger.Log(data, "dns healthcheck")
}

func statusDown(item *types.HealthCheckItem) {
	if item.Status <= 0 {
		item.Status--
		if item.Status < item.DownCount {
			item.Status = item.DownCount
		}
	} else {
		item.Status = -1
	}
}

func statusUp(item *types.HealthCheckItem) {
	if item.Status >= 0 {
		item.Status++
		if item.Status > item.UpCount {
			item.Status = item.UpCount
		}
	} else {
		item.Status = 1
	}
}

func (h *Healthcheck) FilterHealthcheck(qname string, rrset *types.IP_RRSet, mask []int) []int {
	if !h.Enable {
		return mask
	}
	min := rrset.HealthCheckConfig.DownCount
	for i, x := range mask {
		if x == types.IpMaskWhite {
			status := h.redisStat.GetHealthStatus(qname, rrset.Data[i].Ip.String())
			if status > min {
				min = status
			}
		}
	}
	// logger.Default.Debugf("min = %d", min)
	if min < rrset.HealthCheckConfig.UpCount-1 && min > rrset.HealthCheckConfig.DownCount {
		min = rrset.HealthCheckConfig.DownCount + 1
	}
	// logger.Default.Debugf("min = %d", min)
	for i, x := range mask {
		if x == types.IpMaskWhite {
			// logger.Default.Debug("qname: ", rrset.Data[i].Ip.String(), " status: ", h.getStatus(qname, rrset.Data[i].Ip))
			if h.redisStat.GetHealthStatus(qname, rrset.Data[i].Ip.String()) < min {
				mask[i] = types.IpMaskBlack
			}
		} else {
			mask[i] = types.IpMaskBlack
		}
	}
	return mask
}

func (h *Healthcheck) Transfer() {
	itemsEqual := func(item1 *types.HealthCheckItem, item2 *types.HealthCheckItem) bool {
		if item1 == nil || item2 == nil {
			return false
		}
		if item1.Ip != item2.Ip || item1.Uri != item2.Uri || item1.Port != item2.Port ||
			item1.Protocol != item2.Protocol || item1.Enable != item2.Enable ||
			item1.UpCount != item2.UpCount || item1.DownCount != item2.DownCount || item1.Timeout != item2.Timeout {
			return false
		}
		return true
	}

	limiter := time.Tick(time.Millisecond * 50)
	for {
		domains := h.redisData.GetZones()
		for _, domain := range domains {
			domainId := h.getDomainId(domain)
			subdomains := h.redisData.GetZoneLocations(domain)
			for _, subdomain := range subdomains {
				select {
				case <-h.quit:
					h.quitWG.Done()
					return
				case <-limiter:
					record, err := h.redisData.GetLocation(domain, subdomain)
					if err != nil {
						logger.Default.Errorf("cannot get location : zone -> %s, location -> %s", domain, subdomain)
						continue
					}
					for _, rrset := range []*types.IP_RRSet{&record.A, &record.AAAA} {
						if !rrset.HealthCheckConfig.Enable {
							continue
						}
						for i := range rrset.Data {
							key := record.Fqdn + ":" + rrset.Data[i].Ip.String()
							newItem := &types.HealthCheckItem{
								Ip:        rrset.Data[i].Ip.String(),
								Port:      rrset.HealthCheckConfig.Port,
								Host:      record.Fqdn,
								Enable:    rrset.HealthCheckConfig.Enable,
								DownCount: rrset.HealthCheckConfig.DownCount,
								UpCount:   rrset.HealthCheckConfig.UpCount,
								Timeout:   rrset.HealthCheckConfig.Timeout,
								Uri:       rrset.HealthCheckConfig.Uri,
								Protocol:  rrset.HealthCheckConfig.Protocol,
								DomainId:  domainId,
							}
							oldItem, err := h.redisStat.GetHealthcheckItem(key)
							if err != nil || !itemsEqual(oldItem, newItem) {
								h.redisStat.SetHealthcheckItem(newItem)
							}
							h.redisStat.SetHealthcheckItemExpiration(key, h.updateInterval)
						}
					}
				}
			}
		}
	}
}
