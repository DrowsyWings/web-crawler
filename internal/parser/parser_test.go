package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)


func TestParserHTML(t *testing.T){
	html := ` 
		<html>
		<head>
			<title>Test Page</title>
            <meta name="description" content="Just testing">
            <meta name="keywords" content="test,test,test">
		</head>
		<body>
			Hello World
			<a href="/test1">test 1</a>
			<a href="https://example.com/test2">test 2</a>
			<a href="https://test3.com/">test3</a>
		</body>
		</html>
		`
	baseUrl:= "https://example.com"
	result,err := ParseHTML(baseUrl,html)

	assert.NoError(t,err)


	assert.Equal(t,"Test Page",result.Title)
	assert.Equal(t, "Just testing",result.Description)
	assert.Equal(t,"test,test,test",result.Keywords)

	expectedLinks := []string{
		"https://example.com/test1",
		"https://example.com/test2",

	}

	assert.ElementsMatch(t,expectedLinks,result.Links)



}