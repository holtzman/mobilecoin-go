package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Since there's some compile bugs for libmobilecoind in OSX, the tests can't pass in MAC
func TestFogAddress(t *testing.T) {
	assert := assert.New(t)
	err := ValidateAddress("2v47hNBvaA8dZC7ZMmSP3GpLMgRzN6YwWL1nY39chuDLjGokW3Bwmq3NtWoufizk21mYRJfJmyM12RCdgbsv6CAqXgfBKaA2J6boeRAcVB2")
	assert.Nil(err)

	err = ValidateAddress("3j6VPEb37ZLHe6tYZpqTiVXnk5CrfQhELwhgGawXe1MH9tYRm2oty6FaWdiG6hKcn9qnqdCpxmzYNt1GRHprk7sAyfLJjoYvDHWD9f74K8ZijhkHvpAsE9Hko16EymgDgxedGYc6P31wMzuPxDhkrqnsvqQJ83vsCnTtgZWLix1qoYwJ9BukrbCMCfsN8CTyjFeY9HYRGUrf6u2RfJDU4yftuKz7LwECHw6HJYsJHMj7tP")
	assert.Nil(err)

	err = ValidateAddress("96jVegMejVRKrXzD14wKS7g34c2VbJQ5eNZVNwc5GNBP6qiCb37kpiUJK33F1DJvBmiMoLNBnrjFxTJMUCQo9RbdXZVvudyPC8n458NfaNhSsp7UW4gSbEKsDb5D2uxHxMUqTHTkwpbj2EqHXujaTkGVa7dvfgLTWp4NyQr6VqomHPJaoyzrcpXJHVWmCGz1pmBBnqfWTMq31sakbqFFMrVonjNJFtB4dqR4E1nqcTkk4S")
	assert.Nil(err)

	err = ValidateAddress("gK4GSmZ7de6pCnsPnkqdwV47a14fR7M5JuKf3TAfqwGcsDsa3cJW5gKnu52ZDWepfEPi5T6a55gVGB6AvQKtKKtBQEYSwUpDTpSKfG9Et4QA9zLUQyEcpfCx1t79tuoe93sUezp9wXeyT9eSgQPiMmNBXGx7JfhZXqjmiXGRrDyEMMpgY5B7pWMXzD7SH7UW5AFwJkYoNSL9Ff6N4ztebFGS46H9hJ6VQuGh4wHcDbhY6sGGuJH")
	assert.NotNil(err)

	err = ValidateAddress("PioPgoqXcixip4A59Sqncq3PbUAZ47CRT5AtGWYzy4KD9PnqdxGnrUx6pPHvKp7hZaSQui6bdfKVpgfe325y4zbte2a1fsoerAGsjaY4fsK9j1SsywvUN2smRbY3vGDHxyaCkvvqAhQNHLS9hNgzUxXZaBwtHX1oC2BWGE67oh3gXzbiY1G1gYatwt8pfRNjuGcyJUJgD7Q4mkx7DafKAQLAKT9JVmzQpdaPEZvEDzDcEGLn")
	assert.NotNil(err)
}
