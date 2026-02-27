import { Component } from 'react';

class Counter extends Component {
  state = { count: 0 };

  constructor(props) {
    super(props);
    this.handleClick = this.handleClick.bind(this);
  }

  handleClick() {
    this.setState({ count: this.state.count + 1 });
  }

  render() {
    return null;
  }
}

export default Counter;
