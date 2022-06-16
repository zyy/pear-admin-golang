package clientIP

import (
	"github.com/oschwald/geoip2-golang"
	"net"
)

func GetCityByIP(ipAddr string) (string, error) {
	if ipAddr == "127.0.0.1" || ipAddr == "::1" {
		return "本地内网", nil
	}
	db, err := geoip2.Open("database/GeoLite2-City.mmdb")
	if err != nil {
		return "", err
	}
	defer db.Close()
	ip := net.ParseIP(ipAddr)
	record, err := db.City(ip)
	if err != nil {
		return "", err
	}
	if len(record.City.Names) > 0 {
		if record.City.Names["zh-CN"] == "" && len(record.Subdivisions) > 0 {
			return record.Subdivisions[0].Names["zh-CN"], nil
		} else {
			return record.City.Names["zh-CN"], nil
		}
	} else {
		return "未知地址", nil
	}
}
