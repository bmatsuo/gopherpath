##Getting started with gopherpath

Before starting you need to have a custom domain (to use in your import paths)
pointing to a simple "hello world" application on App Engine.  Consult the Go
[tutorial](https://developers.google.com/appengine/docs/go/gettingstarted/introduction)
and the documentation on [custom
domains](https://developers.google.com/appengine/docs/domain) for help doing
this..

Once you are serving HTTP traffic over your custom domain you can deploy
gopherpath to that application.

For this guide, assume "go.example.com" is the custom domain serving your
application is being served from.

##Deploying gopherpath

- Clone (or fork) the gopherpath repository.

- Modify the app.yaml file so that it references your application and commit the change.

- Deploy the application with `goapp deploy`.

##Configuring gopherpath

Gopherpath is configured through entities in the App Engine datastore, using
the authentication and UI of the App Engine admin dashboard.  In order to
initialize the datastore an http request to your custom domain must be processed
Simply make a GET request to your domain's root path.

    curl -i http://go.example.com

This will return an error (404 Not Found) and give instructions specifying
datastore entities that need modification and how they need to be modified.
After modifying the specified entities repeating the previous curl should
succeed (200 OK).

##Off to the races

That's it. You should now be able to `go get` your custom import paths.

    go get go.example.com/ex
