/*
Copyright (C) 2023-2026 QuantumNous

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
import DOMPurify from 'dompurify'
import { useMemo } from 'react'

import { isInlineSvgLogo } from '@/lib/logo'
import { cn } from '@/lib/utils'

type BrandImageProps = {
  src: string
  alt: string
  className?: string
}

const svgSanitizeOptions = {
  USE_PROFILES: { svg: true, svgFilters: true },
  FORBID_TAGS: ['foreignObject', 'script', 'style'],
  FORBID_ATTR: ['style'],
}

export function BrandImage({ src, alt, className }: BrandImageProps) {
  const svg = useMemo(() => {
    if (!isInlineSvgLogo(src)) return null
    return DOMPurify.sanitize(src, svgSanitizeOptions)
  }, [src])

  if (svg) {
    return (
      <span
        role='img'
        aria-label={alt}
        className={cn(
          'block shrink-0 overflow-hidden [&>svg]:block [&>svg]:size-full',
          className
        )}
        // eslint-disable-next-line react/no-danger -- SVG is sanitized above.
        dangerouslySetInnerHTML={{ __html: svg }}
      />
    )
  }

  return <img src={src} alt={alt} className={className} />
}
