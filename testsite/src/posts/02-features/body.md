### Posts are directories ###
Every post is a directory located under the configured posts directory
(by default, this is `posts`).  This allows a single post to contain multiple
resources, such as images or other assets.  For example, the following image
is contained in this post''s directory.

![Go Gopher]({{link "frontpage.png"}})

### Linking ###
It is easy to link to other content in a Ghostwriter site.  Given the ID of a
post, any other post may link to it via the syntax `{{`{{link "<id>"}}`}}`.

To link to a subresource, use the syntax `{{`{{link "<id>/<resource>"}}`}}`.

Links are evaluated before Markdown, so it is easy to add the previous
notation to your Markdown templates.

For example, here is a link to [the first post]({{link "01-hello-world"}}).
