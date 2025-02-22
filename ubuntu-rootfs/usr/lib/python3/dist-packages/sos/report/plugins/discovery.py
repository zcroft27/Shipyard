# Copyright (C) 2021 Red Hat, Inc., Jose Castillo <jcastillo@redhat.com>

# This file is part of the sos project: https://github.com/sosreport/sos
#
# This copyrighted material is made available to anyone wishing to use,
# modify, copy, or redistribute it subject to the terms and conditions of
# version 2 of the GNU General Public License.
#
# See the LICENSE file in the source distribution for further information.

from sos.report.plugins import Plugin, RedHatPlugin


class Discovery(Plugin, RedHatPlugin):

    short_desc = 'Discovery inspection and reporting tool'
    plugin_name = 'discovery'
    packages = ('discovery', 'discovery-tools',)
    containers = ('dsc-db', 'discovery', 'discovery-toolbox')

    def setup(self):
        self.add_copy_spec([
            "/root/discovery/db/volume/data/userdata/pg_log/",
            "/root/discovery/server/volumes/log/app.log",
            "/root/discovery/server/volumes/log/discovery-server.log",
            "/var/lib/containers/storage/volumes/dsc-data/_data/userdata/log/",
            "/var/discovery/server/volumes/log/",
        ])

        self.add_container_logs(list(self.containers))
# vim: set et ts=4 sw=4 :
