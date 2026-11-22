//go:build darwin && cgo

#import <Cocoa/Cocoa.h>

void setDockIcon(const void *bytes, int n) {
	NSData *data = [NSData dataWithBytes:bytes length:n];
	dispatch_async(dispatch_get_main_queue(), ^{
		NSImage *img = [[NSImage alloc] initWithData:data];
		if (img != nil) {
			[[NSApplication sharedApplication] setApplicationIconImage:img];
		}
	});
}
