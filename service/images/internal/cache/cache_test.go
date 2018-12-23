package cache

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {

	c := New(10, 5, time.Second, time.Minute)
	defer c.Close()

	require.Equal(t,
		&Cache{
			maxFileSize:    10,
			lifetime:       time.Second,
			cleanerTimeout: time.Minute,
			payload:        make([]*cacheItem, 0, 5),
			closed:         0,
		}, c)
}

func TestAddGet(t *testing.T) {

	const Lifetime = time.Second * 10

	c := New(10, 2, Lifetime, time.Minute)
	defer c.Close()

	c.Add("k1", []byte{0x01})
	c.Add("k2", []byte{0x02})

	c.mu.Lock()
	require.Len(t, c.payload, 2)
	expK1 := c.payload[0].Expired
	expK2 := c.payload[1].Expired
	c.mu.Unlock()

	func() {
		// test: source state
		c.mu.Lock()
		defer c.mu.Unlock()

		require.WithinDuration(t, time.Now().Add(Lifetime), expK1, time.Second)
		require.WithinDuration(t, time.Now().Add(Lifetime), expK2, time.Second)

		require.Equal(t,
			[]*cacheItem{
				{
					Key:     "k1",
					Data:    []byte{0x01},
					Expired: expK1,
				},
				{
					Key:     "k2",
					Data:    []byte{0x02},
					Expired: expK2,
				},
			},
			c.payload)
	}()

	{
		// test: get
		val, ok := c.Get("k2")
		require.True(t, ok)
		require.Equal(t, []byte{0x02}, val)
	}

	func() {
		// test: after get
		c.mu.Lock()
		defer c.mu.Unlock()

		require.Equal(t,
			[]*cacheItem{
				{
					Key:     "k2",
					Data:    []byte{0x02},
					Expired: expK2,
				},
				{
					Key:     "k1",
					Data:    []byte{0x01},
					Expired: expK1,
				},
			},
			c.payload)
	}()

	func() {
		// test: overflow cache
		c.Add("k3", []byte{0x03})

		c.mu.Lock()
		defer c.mu.Unlock()

		require.Len(t, c.payload, 2)
		expK1 = c.payload[0].Expired
		expK2 = c.payload[1].Expired

		// test: replace last item
		require.Equal(t,
			[]*cacheItem{
				{
					Key:     "k2",
					Data:    []byte{0x02},
					Expired: expK1,
				},
				{
					Key:     "k3",
					Data:    []byte{0x03},
					Expired: expK2,
				},
			},
			c.payload)
	}()
}

func TestAutoCleaner(t *testing.T) {

	{
		// test: remove from start
		const Lifetime = time.Second

		c := New(10, 2, Lifetime, Lifetime)
		defer c.Close()

		c.Add("k1", []byte{0x01})
		time.Sleep(Lifetime / 2)
		c.Add("k2", []byte{0x02})

		func() {
			// before clean
			c.mu.Lock()
			defer c.mu.Unlock()

			require.Len(t, c.payload, 2)

			require.Equal(t,
				[]*cacheItem{
					{
						Key:     "k1",
						Data:    []byte{0x01},
						Expired: c.payload[0].Expired,
					},
					{
						Key:     "k2",
						Data:    []byte{0x02},
						Expired: c.payload[1].Expired,
					},
				},
				c.payload)
		}()

		time.Sleep(Lifetime / 2)

		func() {
			// after clean
			c.mu.Lock()
			defer c.mu.Unlock()

			require.Len(t, c.payload, 1)

			require.Equal(t,
				[]*cacheItem{
					{
						Key:     "k2",
						Data:    []byte{0x02},
						Expired: c.payload[0].Expired,
					},
				},
				c.payload)
		}()
	}

	{
		// test: remove from middle
		const Lifetime = time.Second

		c := New(10, 3, Lifetime, Lifetime)
		defer c.Close()

		c.Add("k1", []byte{0x01})
		time.Sleep(Lifetime / 2)
		c.Add("k2", []byte{0x02})
		c.Add("k3", []byte{0x03})

		// move secondary to start
		c.Get("k2")

		func() {
			// before clean
			c.mu.Lock()
			defer c.mu.Unlock()

			require.Len(t, c.payload, 3)

			require.Equal(t,
				[]*cacheItem{
					{
						Key:     "k2",
						Data:    []byte{0x02},
						Expired: c.payload[0].Expired,
					},
					{
						Key:     "k1",
						Data:    []byte{0x01},
						Expired: c.payload[1].Expired,
					},
					{
						Key:     "k3",
						Data:    []byte{0x03},
						Expired: c.payload[2].Expired,
					},
				},
				c.payload)
		}()

		time.Sleep(Lifetime / 2)

		func() {
			// after clean
			c.mu.Lock()
			defer c.mu.Unlock()

			require.Len(t, c.payload, 2)

			require.Equal(t,
				[]*cacheItem{
					{
						Key:     "k2",
						Data:    []byte{0x02},
						Expired: c.payload[0].Expired,
					},
					{
						Key:     "k3",
						Data:    []byte{0x03},
						Expired: c.payload[1].Expired,
					},
				},
				c.payload)
		}()
	}
}

func TestMultiThreads(t *testing.T) {

	c := New(10, 3, time.Millisecond*10, time.Second/10)
	defer c.Close()

	data := []byte{0x01}

	for thread := 0; thread < 1000; thread++ {
		go func() {

			for i := 0; i < 100; i++ {
				key := strconv.Itoa(i % 3)
				if i%2 == 0 {
					c.Add(key, data)
				} else {
					c.Get(key)
				}
			}
		}()
	}
}
