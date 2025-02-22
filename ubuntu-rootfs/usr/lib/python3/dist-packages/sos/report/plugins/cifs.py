# This file is part of the sos project: https://github.com/sosreport/sos
#
# This copyrighted material is made available to anyone wishing to use,
# modify, copy, or redistribute it subject to the terms and conditions of
# version 2 of the GNU General Public License.
#
# See the LICENSE file in the source distribution for further information.

from sos.report.plugins import Plugin, IndependentPlugin


class Cifs(Plugin, IndependentPlugin):

    short_desc = 'SMB file system information'
    plugin_name = 'cifs'
    profiles = ('storage', 'network', 'cifs')
    packages = ('cifs-utils',)

    def setup(self):
        self.add_forbidden_path([
            "/proc/fs/cifs/traceSMB",
            "/proc/fs/cifs/cifsFYI",
        ])

        self.add_copy_spec([
            "/etc/request-key.d/cifs.spnego.conf",
            "/etc/request-key.d/cifs.idmap.conf",
            "/proc/keys",
            "/proc/fs/cifs/*",
        ])

# vim: set et ts=4 sw=4 :
