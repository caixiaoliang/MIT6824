package mapreduce

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"unsafe"
)

func BytesToString(b []byte) string {
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	sh := reflect.StringHeader{bh.Data, bh.Len}
	return *(*string)(unsafe.Pointer(&sh))
}

func doMap(
	jobName string, // the name of the MapReduce job
	mapTask int, // which map task this is
	inFile string, //this just is a file name!
	nReduce int, // the number of reduce task that will be run ("R" in the paper)
	mapF func(filename string, contents string) []KeyValue,
) {

	//Step1 first read the file and transfer it to users function mapF
	//The map function is called once for each file of input

	contents, errOnReadFile := ioutil.ReadFile(inFile)
	if errOnReadFile != nil {
		log.Printf("read file %s failed", inFile)
	}

	contents_result := BytesToString(contents)
	//open input file read and  send file contents to map


	// 首先把该worker 接收到的文件用用户输入的map方法处理，返回一个k,v数组
	mapResKeyValueArr := mapF(inFile, contents_result)

	//store encorder
	var encorders = make([]*json.Encoder, nReduce)

	//create files
	// 分区和排序
	// Partitioner的作用：
	// 对map端输出的数据key作一个散列，使数据能够均匀分布在各个reduce上进行后续操作，避免产生热点区。

	// 它以key的Hash值对 Reducer 的数目取模，得到对应的Reducer。这样保证如果有相同的key值，
	// 肯定被分配到同一个reducre上。如果有N个reducer，编号就为0,1,2,3……(N-1)。

	//每个worker都会接收到一个文件输入，首先对该输入的文件用 用户输入的map方法处理，返回一个k,v数组
	//然后再生成n个文件，这里n就是reduce的个数
	//比如有N个reduce，这里就把输入分成n份
	//map 函数用来处理文件，得到的是一个文件的内容，返回的是K,v形式的数据，
	//我们把数据根据key进行hash 然后对reduce数取模，分片到不同的reduce文件中去
	//到这里map的任务就结束了
	//map可以保证相同的key的数据肯定会被打到同一个reduce上
	//这里为甚么会出现多个key分配到同一个reduce的场景？
	//因为是先对 key hash，然后再取模，取模的过程就会导致多个key落到同一个reduce上
	//但是 同一个key的所有数据肯定会在同一个reduce上的，因为 key.hash%8 肯定在一个reduce上

	for r := 0; r < nReduce; r++ {
		intermediateFileName := reduceName(jobName, mapTask, r)

		filePtr, errOnCreate := os.Create(intermediateFileName)
		defer filePtr.Close()

		if errOnCreate != nil {
			fmt.Println("创建文件失败，err=", errOnCreate)
			return
		}
		encorders[r] = json.NewEncoder(filePtr)
	}

	//reduceTaskInt 和encorders 中文件句柄的下标记是对应的
	for _, kv := range mapResKeyValueArr {
		reduceTaskInt := ihash(kv.Key) % nReduce

		errOnEncode := encorders[reduceTaskInt].Encode(&kv)

		if errOnEncode != nil {
			fmt.Println("编码失败，err=", errOnEncode)
		} else {
			// fmt.Println("编码成功")
		}
	}

	//Call ihash() (see
	// below) on each key, mod nReduce, to pick r for a key/value pair.

	//
	// doMap manages one map task: it should read one of the input files
	// (inFile), call the user-defined map function (mapF) for that file's
	// contents, and partition mapF's output into nReduce intermediate files.
	//
	// There is one intermediate file per reduce task. The file name
	// includes both the map task number and the reduce task number. Use
	// the filename generated by reduceName(jobName, mapTask, r)
	// as the intermediate file for reduce task r. Call ihash() (see
	// below) on each key, mod nReduce, to pick r for a key/value pair.
	//
	// mapF() is the map function provided by the application. The first
	// argument should be the input file name, though the map function
	// typically ignores it. The second argument should be the entire
	// input file contents. mapF() returns a slice containing the
	// key/value pairs for reduce; see common.go for the definition of
	// KeyValue.
	//
	// Look at Go's ioutil and os packages for functions to read
	// and write files.
	//
	// Coming up with a scheme for how to format the key/value pairs on
	// disk can be tricky, especially when taking into account that both
	// keys and values could contain newlines, quotes, and any other
	// character you can think of.
	//
	// One format often used for serializing data to a byte stream that the
	// other end can correctly reconstruct is JSON. You are not required to
	// use JSON, but as the output of the reduce tasks *must* be JSON,
	// familiarizing yourself with it here may prove useful. You can write
	// out a data structure as a JSON string to a file using the commented
	// code below. The corresponding decoding functions can be found in
	// common_reduce.go.
	//
	//   enc := json.NewEncoder(file)
	//   for _, kv := ... {
	//     err := enc.Encode(&kv)
	//
	// Remember to close the file after you have written all the values!
	//
	// Your code here (Part I).
	//
}

func ihash(s string) int {
	h := fnv.New32a()
	h.Write([]byte(s))
	return int(h.Sum32() & 0x7fffffff)
}
