mockgen -source=pkg/files/store.go -destination=pkg/files/store_mock.go -package=files
mockgen -source=pkg/sneatv/tabs_app.go -destination=pkg/sneatv/tabs_app_mock.go -package=sneatv
mockgen -source=pkg/sneatv/tabs_app.go -destination=pkg/sneatv/tabs_app_mock.go -package=sneatv
mockgen -source=pkg/viewers/previewer.go -destination=pkg/viewers/previewer_mock.go -package=viewers
mockgen -source=pkg/viewers/dir_previewer_app.go -destination=pkg/viewers/dir_previewer_app_mock.go -package=viewers