package homework02

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// Q1: 指针 - 增加值
// 编写一个函数，该函数接收一个整数指针作为参数，在函数内部将该指针指向的值增加10
func AddTen(val *int) {
	*val += 10
}

// Q2: 指针 - 切片元素乘2
// 实现一个函数，接收一个整数切片的指针，将切片中的每个元素乘以2
func DoubleSlice(slice *[]int) {
	if slice == nil {
		return
	}

	for i := range *slice {
		(*slice)[i] *= 2
	}
}

// Q3: Goroutine - 奇偶数打印
// 编写一个程序，使用 go 关键字启动两个协程，一个协程打印从1到10的奇数，另一个协程打印从2到10的偶数。
// 提示：为了测试能看到输出，可以使用 sync.WaitGroup 确保主程序等待协程结束
func PrintOddEven() {
	// 1、定义sync，确保线程阻塞，输出所有结果
	var wg sync.WaitGroup
	wg.Add(2)

	// 2、启动第一个协程打印1到10的奇数
	go func() {
		defer wg.Done()
		fmt.Println("输出奇数序列：")
		for i := 1; i < 10; i += 2 {
			fmt.Println(i)
		}
	}()

	// 3、启动第二个协程打印2到10的偶数
	go func() {
		defer wg.Done()
		fmt.Println("输出偶数序列：")
		for i := 2; i <= 10; i += 2 {
			fmt.Println(i)
		}
	}()

	// 4、阻塞主线程，直至全部输出完毕
	wg.Wait()

	fmt.Println("所有序列输出完毕！")
}

// Q4: Goroutine - 任务调度
// 设计一个任务调度器，接收一组任务（可以用函数表示），并使用协程并发执行这些任务，同时统计每个任务的执行时间。
func TaskScheduler(tasks []func()) {
	var wg sync.WaitGroup
	// 用于并发安全地存储每个任务的执行时间
	durations := make([]time.Duration, len(tasks))

	for i, task := range tasks {
		wg.Add(1)
		// 将索引和任务函数作为参数传入闭包，避免循环变量共享问题
		go func(index int, fn func()) {
			defer wg.Done()

			start := time.Now()
			fn()                                 // 执行具体的任务函数
			durations[index] = time.Since(start) // 记录耗时

			fmt.Printf("任务 %d 执行完毕，耗时: %v\n", index, durations[index])
		}(i, task)
	}

	wg.Wait() // 等待所有协程执行完毕

	// 输出汇总统计
	fmt.Println("\n 任务执行时间统计汇总:")
	var totalDuration time.Duration
	for i, d := range durations {
		fmt.Printf("任务 %d : %v\n", i, d)
		totalDuration += d
	}
	fmt.Printf("任务累计耗时(串行总和): %v\n", totalDuration)

}

// Q5: 面向对象 - 接口
// 定义一个 Shape 接口，包含 Area() 和 Perimeter() 两个方法。
type Shape interface {
	Area() float64
	Perimeter() float64
}

type Rectangle struct {
	Width  float64
	Height float64
}

// TODO: 为 Rectangle 实现 Shape 接口
func (r *Rectangle) Area() float64 {

	return r.Width * r.Height
}

func (r *Rectangle) Perimeter() float64 {

	return (r.Width + r.Height) * 2
}

type Circle struct {
	Radius float64
}

// TODO: 为 Circle 实现 Shape 接口
func (c *Circle) Area() float64 {

	return math.Pi * c.Radius * c.Radius
}

func (c *Circle) Perimeter() float64 {

	return 2 * math.Pi * c.Radius
}

// Q6: 面向对象 - 组合
// 使用组合的方式创建一个 Person 结构体，包含 Name 和 Age 字段
type Person struct {
	Name string
	Age  int
}

// 再创建一个 Employee 结构体，组合 Person 结构体并添加 EmployeeID 字段
type Employee struct {
	Person
	EmployeeID string
}

// TODO: 为 Employee 结构体实现一个 PrintInfo() string 方法，返回员工的信息 (格式自定，包含Name, Age, EmployeeID)
func (e Employee) PrintInfo() string {
	return fmt.Sprintf("员工信息: 姓名=%s, 年龄=%d, 员工ID=%s",
		e.Name, e.Age, e.EmployeeID)
}

// Q7: Channel - 生产者消费者
// 编写一个程序，使用通道实现两个协程之间的通信。
// 一个协程生成从1到10的整数，并将这些整数发送到通道中，另一个协程从通道中接收这些整数并打印出来。
// 为了方便测试，请将接收到的数字返回
func ProducerConsumer() []int {
	ch := make(chan int)
	var wg sync.WaitGroup

	// 用于存储结果的切片
	results := make([]int, 0, 10)

	// 启动生产者协程
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 1; i <= 10; i++ {
			ch <- i // 将整数发送到通道
		}
		close(ch) // 发送完毕后关闭通道
	}()

	// 启动消费者协程
	wg.Add(1)
	go func() {
		defer wg.Done()
		for num := range ch { // 从通道接收数据直到通道关闭
			results = append(results, num)
		}
	}()

	// 等待两个协程都执行完毕
	wg.Wait()

	return results
}

// Q8: Channel - 缓冲通道
// 实现一个带有缓冲的通道，生产者协程向通道中发送100个整数，消费者协程从通道中接收这些整数并打印。
func BufferedChannel() {
	// 创建一个容量为 10 的带缓冲通道
	bufferSize := 10
	ch := make(chan int, bufferSize)

	// 使用 WaitGroup 确保主协程等待所有子协程执行完毕
	var wg sync.WaitGroup

	// 生产者协程
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(ch) // 生产完毕后关闭通道，通知消费者停止接收

		for i := 1; i <= 100; i++ {
			ch <- i // 向缓冲通道发送数据
			fmt.Printf("生产者发送: %d\n", i)
		}
	}()

	// 消费者协程
	wg.Add(1)
	go func() {
		defer wg.Done()
		// 使用 range 遍历通道，当通道被关闭且为空时，循环会自动结束
		for val := range ch {
			fmt.Printf("消费者接收: %d\n", val)
		}
	}()

	// 等待所有协程完成
	wg.Wait()

	fmt.Println("所有任务执行完毕，程序退出。")
}

// Q9: 锁机制 - Mutex
// 编写一个程序，使用 sync.Mutex 来保护一个共享的计数器。
// 启动10个协程，每个协程对计数器进行1000次递增操作，最后输出计数器的值。
func MutexCounter() int {
	var count int
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 启动 10 个协程
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// 每个协程对计数器进行 1000 次递增操作
			for j := 0; j < 1000; j++ {
				mu.Lock() // 加锁，保护临界区
				count++
				mu.Unlock() // 解锁
			}
		}()
	}

	// 等待所有协程执行完毕
	wg.Wait()

	// 输出最终的计数器值
	fmt.Printf("最终计数器的值: %d\n", count)
	return count
}

// Q10: 锁机制 - Atomic
// 使用原子操作（ sync/atomic 包）实现一个无锁的计数器。
// 启动10个协程，每个协程对计数器进行1000次递增操作，最后输出计数器的值。
func AtomicCounter() int32 {
	var count int32
	var wg sync.WaitGroup

	// 启动 10 个协程
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// 每个协程递增 1000 次
			for j := 0; j < 1000; j++ {
				atomic.AddInt32(&count, 1)
			}
		}()
	}

	// 等待所有协程完成
	wg.Wait()

	// 输出最终计数器的值
	fmt.Println("最终计数器值:", atomic.LoadInt32(&count))
	return count
}
