Rapidos
=======

Rapidos is a minimal Linux OS image generator and runner.

- All dependencies are built from source or obtained from the local system

  - There are **no** precreated or downloaded *magic* images

- u-root[1] is used for initramfs image generation

  - u-root provides a lightweight and pluggable Linux userspace

- It is written in pure Go and easily extendible

- A thin wrapper around QEMU is provided for booting images


Building & Installing
---------------------

Ensure that you have a recent (1.11+) Go version.

Get rapidos::

        go get gitlab.com/rapidos/rapidos

Clone Linux::

        git clone https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git

Build your kernel::

        cd linux
        # replace <rapidos> with your source directory in GOPATH
        cp <rapidos>/tools/vanilla_kernel.config .config
        make -j4
        INSTALL_MOD_PATH=./mods make modules_install


Running
-------

Configure rapidos.conf::

        # replace <rapidos> with your source directory in GOPATH
        cd <rapidos>
        cp rapidos.conf.example rapidos.conf
        # edit rapidos.conf and set KERNEL_SRC

Determine which init image you would like to generate for testing::

        ./rapidos -list

Generate and boot the image, e.g::

        ./rapidos -cut example -boot

Subsequent runs (without -cut) boot the previously generated image.

Some images require a virtual network connection, in which case bridge
and tap interfaces can be provisioned via::

        # set TAP_USER and MAC_ADDR* in rapidos.conf
        sudo tools/br_setup.sh


Extending
---------

To create your own image type, simply copy the example init::

        cp -r inits/example inits/my-new-init

Edit ``inits/my-new-init/manifest.go``, set *Name* and add any Go packages
(*Pkgs*), kernel modules (*Kmods*), binaries (*Bins*) or miscelaneous files
(*Files*) that you wish to have included in your image. *Init* should also be
configured with::

        Init: gitlab.com/rapidos/rapidos/inits/my-new-init

Edit the *Init* source referred to above at ``inits/my-new-init/uinit/main.go``.
It will be executed immediately when your image boots.

Finally, ensure that your init is registered with the main application by
editing ``rapidos.go`` and setting::

        import (
                ...
                // inits registered via AddManifest() callback
                _ "gitlab.com/rapidos/rapidos/inits/my-new-init"
        )

Cut and boot an image with your newly created init::

        go build rapidos.go
        ./rapidos -cut my-new-init -boot


Links
-----

1) u-root project homepage
   http://u-root.tk/
