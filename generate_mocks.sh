mockgen -source=pkg/filetug/navigator/app.go -destination=pkg/filetug/navigator/app_mock_test.go -package=navigator
mockgen -source=pkg/files/store.go -destination=pkg/files/store_mock.go -package=files
mockgen -source=pkg/viewers/previewer.go -destination=pkg/viewers/previewer_mock.go -package=viewers
