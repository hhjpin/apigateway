package routing

import (
	"bytes"
	"testing"
)

func TestMatch(t *testing.T) {
	var input, pattern, backend [][]byte

	input = bytes.Split([]byte("GET@/front/test/123"), UriSlash)
	pattern = bytes.Split([]byte("GET@/front/test/:test_id"), UriSlash)
	backend = bytes.Split([]byte("GET@/backend/test/:test_id"), UriSlash)
	ok, replace := match(input, pattern, backend)
	if !ok {
		t.FailNow()
	} else if !bytes.Equal(replace, []byte("GET@/backend/test/123")) {
		t.FailNow()
	}

	input = bytes.Split([]byte("GET@/front/test/123"), UriSlash)
	pattern = bytes.Split([]byte("GET@/front/*any"), UriSlash)
	backend = bytes.Split([]byte("GET@/backend/test/*any"), UriSlash)
	ok, replace = match(input, pattern, backend)
	if !ok {
		t.FailNow()
	} else if !bytes.Equal(replace, []byte("GET@/backend/test/test/123")) {
		t.FailNow()
	}

	input = bytes.Split([]byte("GET@/front/v1/test/update/123"), UriSlash)
	pattern = bytes.Split([]byte("GET@/front/:version/test/*any"), UriSlash)
	backend = bytes.Split([]byte("GET@/backend/:version/*any"), UriSlash)
	ok, replace = match(input, pattern, backend)
	if !ok {
		t.FailNow()
	} else if !bytes.Equal(replace, []byte("GET@/backend/v1/update/123")) {
		t.FailNow()
	}
}
