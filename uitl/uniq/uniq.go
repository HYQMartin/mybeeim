//给连接进来的服务器一个唯一的id =i，从0开始到2^64-1
package uniq

var (
	num = make(chan uint64)
)

func init() {
	go func() {
		for i := uint64(1000000); ; i++ {
			num <- i
		}
	}()
}

//********产生递增uint64数，注意：重启服务器又会从0开始，应当考虑从数据庫里面读取记录
func GetUniq() uint64 {
	return <-num
}
