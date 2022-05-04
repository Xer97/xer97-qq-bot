package bill

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Bill struct {
	Name  string
	Price int64
}

const (
	rule = "\n@我并输入指令进行账单操作：\n\n> '事项 金额':（如：晚餐 -15）（中间有空格）进行记账，金额负数为支出，正数为收入\n\n> '日期':（如：2022.05.01）统计指定日期收支明细情况"
	// 机器人@信息文本，一般机器人固定了该信息也是固定的，也可以进行动态获取当前机器人id
	botAtInfo = "<@!424268190167377645>"
)

var detailDb = make(map[string]map[int64][]Bill)

func Enter(uid string, content string) string {
	// 解析指令
	// 去除@信息
	content = strings.Replace(content, botAtInfo, "", -1)
	// 去除左右空格
	content = strings.TrimSpace(content)
	// 1、单纯@展示功能提示
	if content == "" {
		return rule
	}
	// 解析指令
	split := strings.Split(content, " ")
	if len(split) == 2 {
		// 2、事项+金额 新增收入/支出 输出当天明细
		price, err := strconv.ParseInt(split[1], 10, 64)
		if err != nil {
			return rule
		}
		insertDb(uid, split[0], price)
		return getTodayDetail(uid)
	} else if len(split) == 1 {
		// 3、日期，输出指定日期明细、累计收入/支出
		tm3, err := time.Parse("2006.01.02", split[0])
		if err != nil {
			return rule
		}
		return getDetailByTime(uid, tm3)
	}
	return rule
}

// getByUid 获取当前用户某天的所有明细，真正流程中需查库，本实现为内存存储，重启后数据丢失
func getByUid(uid string) map[int64][]Bill {
	if detailDb[uid] == nil {
		detailDb[uid] = make(map[int64][]Bill)
	}
	return detailDb[uid]
}

// insertDb 新增数据，真正流程中需落库，本实现为内存存储，重启后数据丢失
func insertDb(uid string, name string, price int64) {
	t := time.Now()
	timeToday := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)

	currUid := getByUid(uid)
	if currUid[timeToday.Unix()] == nil {
		currUid[timeToday.Unix()] = []Bill{}
	}
	bill := new(Bill)
	bill.Name = name
	bill.Price = price
	currUid[timeToday.Unix()] = append(currUid[timeToday.Unix()], *bill)
}

// getTodayDetail 打印今天的明细
func getTodayDetail(uid string) string {
	t := time.Now()
	timeToday := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	return getDetailByTime(uid, timeToday)
}

// getDetailByTime 打印指定日期的明细
func getDetailByTime(uid string, time time.Time) string {
	currUid := getByUid(uid)
	bills := currUid[time.Unix()]
	var ret = fmt.Sprintf("%v 明细如下:\n----------\n", time.Format("2006.01.02"))
	var sum int64 = 0
	var p int64 = 0
	var n int64 = 0
	for _, v := range bills {
		ret = ret + v.Name + " : " + strconv.FormatInt(v.Price, 10) + "\n"
		sum += v.Price
		if v.Price < 0 {
			n -= v.Price
		} else {
			p += v.Price
		}
	}
	ret += "----------\n"
	ret += fmt.Sprintf("净收入:%v, 总收入:%v, 总支出:%v", sum, p, n)
	return ret
}
