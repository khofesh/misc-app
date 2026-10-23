var a = scriptArgs
var doc = Document.openDocument(a[0])
var page = doc.loadPage(0)
page.createLink([+a[2], +a[3], +a[4], +a[5]], a[6])
doc.save(a[1])
