import { Link } from '@inertiajs/react'

export default function Dashboard(props) {
  return (
    <div>
      <div>Hello World : {JSON.stringify(props)}</div>
      <Link href="/">back hello</Link>
    </div>
  )
}
