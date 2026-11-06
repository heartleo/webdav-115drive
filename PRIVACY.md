# Privacy Policy

webdav-115drive is a self-hosted application that provides access to a user's 115 Drive account. This policy describes how the application handles information when you run it. Your hosting provider and 115 have their own privacy practices.

## Information used and shared

The application reads the 115 authentication credentials you provide (`UID`, `CID`, `SEID`, and `KID`) and uses them to authenticate requests to 115. It accesses account-associated directory and file information, including names, paths, identifiers, and download URLs, to list and serve your files. File content is retrieved from 115 or its download servers. Depending on the download strategy, a WebDAV client may receive a download URL directly or the application may proxy the content.

If you confirm a cloud-download submission in the terminal interface, the application sends the supplied link and destination to 115 and may retrieve task status. The resulting task and files are handled by 115.

WebDAV clients can receive directory information and file content available through the configured service. Anyone with access to that service may be able to access this information, subject to its authentication configuration.

## Local configuration and cache

You can provide credentials and WebDAV authentication settings through `config.yaml`, a `.env` file, or environment variables. These are managed by you and may remain on the device or in your deployment environment until you remove them. Protect configuration files and environment variables from unauthorized access.

The application temporarily caches directory information and, in proxy mode, download URLs in process memory. Cache entries expire according to `drive.cache_expire`, which defaults to one minute; the in-memory cache is removed when the process exits. Other software or services in your deployment may retain their own data.

## Logs

In server mode, the application writes logs to standard output. Request logs can include the request method, path, client address, status, and duration. Error and debug logs may include file identifiers, download URLs, or upstream response content. Your terminal, service manager, container runtime, or hosting provider may capture and retain these logs according to its configuration. Treat logs as potentially sensitive.

## Removing information

Stop the application to clear its in-memory cache. Remove credentials and other sensitive settings from your configuration files, environment variables, and deployment secrets, and delete any captured logs or backups according to your environment's procedures. To invalidate 115 credentials, use the account or session management options provided by 115. Information retained by 115, WebDAV clients, or your hosting provider is governed by those services and must be managed separately.

## Contact

For questions about this policy, open an issue at https://github.com/heartleo/webdav-115drive/issues. Do not include credentials, private download URLs, or sensitive logs in an issue.
