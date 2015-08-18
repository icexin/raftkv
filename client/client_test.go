package raftkv

import "testing"

func BenchmarkRead(t *testing.B) {
	cli := NewClient([]string{"127.0.0.1:8000"}, nil)
	err := cli.Write([]byte("hello"), []byte("world"))
	if err != nil {
		t.Fatal(err)
	}

	t.RunParallel(func(pb *testing.PB) {
		cli := NewClient([]string{"127.0.0.1:8000"}, nil)
		for pb.Next() {
			_, err := cli.Read([]byte("hello"))
			if err != nil {
				t.Fatal(err)
			}
		}
		cli.Close()
	})
}

func BenchmarkWrite(t *testing.B) {
	t.RunParallel(func(pb *testing.PB) {
		cli := NewClient([]string{"127.0.0.1:8000"}, nil)
		for pb.Next() {
			err := cli.Write([]byte("hello"), []byte("world"))
			if err != nil {
				t.Fatal(err)
			}
		}
		cli.Close()
	})
}
