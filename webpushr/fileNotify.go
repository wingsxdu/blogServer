package webpushr

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"os"
	"path/filepath"
)

type NotifyFile struct {
	watch *fsnotify.Watcher
	quit  chan int
}

func NewNotifyFile() (*NotifyFile, error) {
	w := new(NotifyFile)
	var err error
	w.watch, err = fsnotify.NewWatcher()
	w.quit = make(chan int)
	return w, err
}

// 监控目录
func (n *NotifyFile) WatchDir(dir string) {
	// 遍历目录下的所有子目录

	// 使用 filepath.Walk() 在目录较大时性能较低，且不会释放文件描述符，容易发生 fcntl: too many open files 错误
	// linux 下进程默认最大 fd 为 1024，目前约为300多个，未来再改吧
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		// 判断是否为目录，若是则监控目录
		if err != nil {
			panic(err)
		}
		if info.IsDir() {
			path, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			err = n.watch.Add(path)
			if err != nil {
				return err
			}
			fmt.Println("监控：", path)
		} else {
		}
		return nil
	})

	go n.WatchEvent()
}

func (n *NotifyFile) WatchEvent() {
	for {
		select {
		case ev := <-n.watch.Events:

			if ev.Op&fsnotify.Create == fsnotify.Create {
				// 获取新创建文件的信息，如果是目录，则加入监控中
				file, err := os.Stat(ev.Name)
				fmt.Println(ev.Name)
				if err == nil && file.IsDir() {
					n.watch.Add(ev.Name)
				}
				if file.Name() == "index.html" { // 有新文章发布，发送推送通知
					err = webpush()
					if err != nil {
						fmt.Println(err)
					}
				}
			}

			if ev.Op&fsnotify.Write == fsnotify.Write {
				continue
			}

			if ev.Op&fsnotify.Remove == fsnotify.Remove {
				// 如果删除文件是目录，则移除监控
				fi, err := os.Stat(ev.Name)
				if err == nil && fi.IsDir() {
					n.watch.Remove(ev.Name)
				}
			}

			if ev.Op&fsnotify.Rename == fsnotify.Rename {
				// 重命名文件或目录，直接 remove
				n.watch.Remove(ev.Name)
			}
			if ev.Op&fsnotify.Chmod == fsnotify.Chmod {
				continue
			}

		case err := <-n.watch.Errors:
			// TODO:更好的错误处理
			fmt.Println("error : ", err)
			return

		case <-n.quit:
			return

		}
	}
}