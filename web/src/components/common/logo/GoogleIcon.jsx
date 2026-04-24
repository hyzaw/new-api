/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React from 'react';

const GoogleIcon = ({ size = 20, className = '', style = {}, ...props }) => {
  const pixelSize = typeof size === 'number' ? `${size}px` : size;

  return (
    <svg
      viewBox='0 0 24 24'
      width={pixelSize}
      height={pixelSize}
      aria-hidden='true'
      focusable='false'
      className={className}
      style={style}
      {...props}
    >
      <path
        fill='#4285F4'
        d='M23.49 12.27c0-.79-.07-1.54-.2-2.27H12v4.3h6.44a5.5 5.5 0 0 1-2.39 3.61v3h3.87c2.27-2.09 3.57-5.18 3.57-8.64Z'
      />
      <path
        fill='#34A853'
        d='M12 24c3.24 0 5.95-1.07 7.93-2.91l-3.87-3c-1.07.72-2.44 1.15-4.06 1.15-3.12 0-5.75-2.11-6.69-4.96H1.31v3.1A11.99 11.99 0 0 0 12 24Z'
      />
      <path
        fill='#FBBC05'
        d='M5.31 14.28A7.2 7.2 0 0 1 4.94 12c0-.79.14-1.56.37-2.28v-3.1H1.31A11.99 11.99 0 0 0 0 12c0 1.93.46 3.76 1.31 5.38l4-3.1Z'
      />
      <path
        fill='#EA4335'
        d='M12 4.77c1.77 0 3.35.61 4.6 1.8l3.45-3.45C17.94 1.14 15.24 0 12 0A11.99 11.99 0 0 0 1.31 6.62l4 3.1c.94-2.85 3.57-4.95 6.69-4.95Z'
      />
    </svg>
  );
};

export default GoogleIcon;
